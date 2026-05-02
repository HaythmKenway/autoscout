package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HaythmKenway/autoscout/internal/ai"
	"github.com/HaythmKenway/autoscout/internal/controller"
	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/internal/scheduler"
	gui_module "github.com/HaythmKenway/autoscout/pkg/gui"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/proxy"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	"gopkg.in/yaml.v3"
)

const (
	host = "0.0.0.0"
	port = "2222"
)

func main() {
	tgt := flag.String("u", "", "Add Host")
	deamon := flag.Bool("d", false, "Run Autoscout in deamon mode")
	cleardb := flag.Bool("reset", false, "Clear All database")
	htt := flag.String("httpx", "", "Run httpx")
	spi := flag.String("spider", "", "Run spider")
	gui := flag.Bool("g", false, "Start GUI")
	sshMode := flag.Bool("ssh", false, "Start sshserver")
	proxyMode := flag.Bool("p", false, "Start decoupled Proxy and AI Fleet")
	proxyPort := flag.String("port", "8081", "Port for the decoupled proxy")
	genCert := flag.Bool("gencert", false, "Generate MITM Root CA certificate")

	flag.Parse()

	if *genCert {
		crt, _, err := proxy.LoadOrCreateCA()
		if err != nil {
			fmt.Printf("Failed to generate CA: %v\n", err)
			return
		}
		crtPath, _ := proxy.GetCAPaths()
		fmt.Printf("Root CA Generated Successfully!\nCommon Name: %s\nPath: %s\n", crt.Subject.CommonName, crtPath)
		return
	}

	controller.Init()

	if *proxyMode {
		// Read settings for upstream proxy
		settingsPath := os.ExpandEnv("$HOME/.config/autoscout/user-config.yaml")
		data, err := os.ReadFile(settingsPath)
		upstream := ""
		if err == nil {
			var config struct {
				Settings struct {
					UpstreamProxy string `yaml:"upstream_proxy"`
				} `yaml:"settings"`
			}
			yaml.Unmarshal(data, &config)
			upstream = config.Settings.UpstreamProxy
		}

		// Start AI Router in background
		go ai.StartRouter()
		// Start Proxy (Blocking)
		addr := net.JoinHostPort("127.0.0.1", *proxyPort)
		if err := proxy.StartProxy(addr, upstream); err != nil {
			localUtils.Logger(fmt.Sprintf("Proxy failed: %v", err), 2)
		}
		return
	}

	if *sshMode {
		sshdeeznuts()
		return
	}

	if *cleardb {
		if err := db.ClearDB(); err != nil {
			localUtils.Logger(fmt.Sprintf("Error clearing DB: %v", err), 2)
		} else {
			localUtils.Logger("Database cleared", 1)
		}
	}

	if *spi != "" {
		controller.Spider(*spi)
	}

	if *htt != "" {
		dbConn, err := db.OpenDatabase()
		if err != nil {
			localUtils.Logger(fmt.Sprintf("Could not open DB for Httpx: %v", err), 2)
		} else {
			httpx.Httpx(dbConn, *htt)
			dbConn.Close()
		}
	}

	if *tgt != "" {
		msg, err := db.AddTarget(*tgt)
		if err != nil {
			localUtils.Logger(msg+" "+err.Error(), 2)
		} else {
			localUtils.Logger(msg, 1)
		}
	}

	if *gui {
		if err := gui_module.LoadGui(); err != nil {
			localUtils.Logger(fmt.Sprintf("GUI failed: %v", err), 2)
			fmt.Printf("Error starting GUI: %v\n", err)
		}
	}

	if *deamon {
		localUtils.Logger("Starting application in deamon mode", 1)
		scheduler.Skibbidi(true)
		select {}
	}
}

func sshdeeznuts() {
	s, err := wish.NewServer(
		wish.WithAddress(net.JoinHostPort(host, port)),
		wish.WithHostKeyPath(".ssh/id_ed25519"),
		wish.WithMiddleware(
			bubbletea.Middleware(gui_module.SShHandler),
			activeterm.Middleware(),
		),
	)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Could not start server: %v", err), 2)
		return
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	localUtils.Logger("Running over ssh with"+fmt.Sprintf("\n  ssh %s -p %s\n\n", host, port), 1)

	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			localUtils.Logger(fmt.Sprintf("SSH Server failed: %v", err), 2)
			done <- nil
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
		localUtils.Logger(fmt.Sprintf("Could not stop server: %v", err), 1)
	}
}
