package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/HaythmKenway/autoscout/internal/db"
)

func main() {
	tgt := flag.String("u", "", "Add Host")
	deamon := flag.Bool("d", false, "Run Autoscout in deamon mode")
	cleardb := flag.Bool("reset", false, "Clear All database")
	flag.Parse()

	if *cleardb {
		db.ClearDB()
	}
	if *tgt != "" {
		db.AddTarget(*tgt)
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
