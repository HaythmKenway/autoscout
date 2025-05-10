package main

import (
	"flag"
	"fmt"
	"time"
	"context"
	"errors"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/HaythmKenway/autoscout/internal/controller"
	"github.com/HaythmKenway/autoscout/internal/db"
	gui_module "github.com/HaythmKenway/autoscout/pkg/gui"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/server"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	"github.com/charmbracelet/wish/bubbletea"
	// "github.com/charmbracelet/log"

)

const (
	host = "0.0.0.0"
	port = "2222"
)


func main() {
	tgt := flag.String("u", "", "Add Host")
	servermode := flag.Bool("s", false, "Run Autoscout in server mode")
	deamon := flag.Bool("d", false, "Run Autoscout in deamon mode")
	cleardb := flag.Bool("reset", false, "Clear All database")
	htt := flag.String("httpx", "", "Run httpx")
	spi := flag.String("spider", "", "Run spider")
	gui := flag.Bool("g", false, "Start GUI")
	ssh := flag.Bool("ssh",true,"Start sshserver")
	flag.Parse()
	controller.Init()
	if *ssh{
		sshdeeznuts()
	}
	if *cleardb {
		db.ClearDB()
	}
	if *spi != "" {
		controller.Spider(*spi)
	}
	if *htt != "" {
		httpx.Httpx(*htt)
	}
	if *tgt != "" {
		db.AddTarget(*tgt)
	}
	if *gui {
		gui_module.LoadGui()
		localUtils.Logger("hello", 1)
	}
	if *servermode {
		server.Server()
	}
	if *deamon {
		localUtils.Logger("Starting application in deamon mode", 1)

		for true {
			fmt.Println("running as deamon")
			StartUp()
			fmt.Println("next job in ", time.Hour/2)
			time.Sleep(time.Hour / 2)
		}
	}
	return
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
		localUtils.Logger(fmt.Sprintf("Could not start server", "error", err),2)
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)
	localUtils.Logger("Running over ssh with"+fmt.Sprintf("\n  ssh %s -p %s\n\n", host, port),1)
	go func() {
		if err = s.ListenAndServe(); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
			done <- nil
		}
	}()

	<-done
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil && !errors.Is(err, ssh.ErrServerClosed) {
				localUtils.Logger(fmt.Sprintf("Could not stop server", "error", err),1)
	}
}
func StartUp() {
	db.Deamon()
}
