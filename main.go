package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/httpx"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/server"
)

func main() {
	tgt := flag.String("u", "", "Add Host")
	servermode := flag.Bool("s", false, "Run Autoscout in server mode")
	deamon := flag.Bool("d", false, "Run Autoscout in deamon mode")
	cleardb := flag.Bool("reset", false, "Clear All database")
	htt := flag.String("httpx", "", "Run httpx")
	flag.Parse()
	if *cleardb {
		db.ClearDB()
	}
	if *htt != "" {
		httpx.Httpx(*htt)
	}
	if *tgt != "" {
		db.AddTarget(*tgt)
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
func StartUp() {
	db.Deamon()
}
