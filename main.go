package main

import (
	"flag"
	"fmt"
	"time"
	"github.com/HaythmKenway/autoscout/server"
	"github.com/HaythmKenway/autoscout/internal/db"
)

func main() {
	tgt := flag.String("u", "", "Add Host")
	servermode := flag.Bool("s", false, "Run Autoscout in server mode")
	deamon := flag.Bool("d", false, "Run Autoscout in deamon mode")
	cleardb := flag.Bool("reset", false, "Clear All database")
	flag.Parse()
	if *cleardb {
		db.ClearDB()
	}
	if *tgt != "" {
		db.AddTarget(*tgt)
	}
	if *servermode {
		server.Server()
	}
	if *deamon {
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
