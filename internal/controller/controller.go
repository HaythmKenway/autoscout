package controller

import (
	"fmt"

	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/spider"
)

func Init() {
	// CheckTables manages its own connection internally
	db.CheckTables()
}

func Spider(domain string) {
	// 1. Run the Spider Tool (Network operation)
	targets, err := spider.Spider(domain)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Spider execution failed for %s: %v", domain, err), 2)
		return
	}

	// 2. Open Database Connection
	// We open it here to pass it to the db function, ensuring thread safety
	database, err := db.OpenDatabase()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Failed to open database: %v", err), 2)
		return
	}
	defer database.Close()

	// 3. Save Results
	// Pass the 'database' pointer as the first argument
	err = db.AddSpiderTargets(database, domain, targets)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Failed to save spider results: %v", err), 2)
	}
}
