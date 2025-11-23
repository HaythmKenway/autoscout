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

	"github.com/HaythmKenway/autoscout/internal/controller"
	"github.com/HaythmKenway/autoscout/internal/db"
	gui_module "github.com/HaythmKenway/autoscout/pkg/gui"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"

	// Import the scheduler if you want to use the new daemon logic
	// "github.com/HaythmKenway/autoscout/scheduler"

	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
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

	// FIXED: Renamed variable to avoid collision with 'ssh' package
	sshMode := flag.Bool("ssh", false, "Start sshserver")

	flag.Parse()

	// Initialize Controllers/DB Check
	controller.Init()

	if *sshMode {
		sshdeeznuts()
		// Usually SSH server blocks, so we return after it closes
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
		// Controller handles its own DB connection
		controller.Spider(*spi)
	}

	if *htt != "" {
		// FIXED: Httpx now requires a DB connection
		dbConn, err := db.OpenDatabase()
		if err != nil {
			localUtils.Logger(fmt.Sprintf("Could not open DB for Httpx: %v", err), 2)
		} else {
			httpx.Httpx(dbConn, *htt)
			dbConn.Close()
		}
	}

	if *tgt != "" {
		// AddTarget manages its own connection
		msg, err := db.AddTarget(*tgt)
		if err != nil {
			localUtils.Logger(msg+" "+err.Error(), 2)
		} else {
			localUtils.Logger(msg, 1)
		}
	}

	if *gui {
		gui_module.LoadGui()
	}

	if *deamon {
		localUtils.Logger("Starting application in deamon mode", 1)

		// OPTION A: Use your old loop (Legacy)
		for {
			fmt.Println("running as deamon")
			StartUp()
			fmt.Println("next job in ", time.Hour/2)
			time.Sleep(time.Hour / 2)
		}

		// OPTION B: Use your new Scheduler (Recommended)
		/*
			scheduler.Skibbidi(true)
			// Block forever
			select {}
		*/
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

func StartUp() {
	db.Deamon()
}
