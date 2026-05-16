package db

import (
	"database/sql"
	"os/exec"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	_ "github.com/mattn/go-sqlite3"
)

var DatabaseFile = localUtils.GetWorkingDirectory() + "/autoscout.db"

func ClearDB() error {
	cmd := exec.Command("rm", DatabaseFile)
	return cmd.Run()
}

// OpenDatabase is exported so the Scheduler can use it
func OpenDatabase() (*sql.DB, error) {
	dsn := DatabaseFile + "?_busy_timeout=5000"
	return sql.Open("sqlite3", dsn)
}

func Deamon() {
	db, err := OpenDatabase()
	if err != nil {
		localUtils.CheckError(err)
		return
	}
	defer db.Close()

	if err := createTargetTableIfNotExists(db); err != nil {
		localUtils.CheckError(err)
		return
	}

	// Updated to pass db connection
	urls, err := GetTargetsFromTable(db)
	if err != nil {
		localUtils.CheckError(err)
		return
	}

	rateLimit := localUtils.GetRateLimit()
	for _, url := range urls {
		// SubdomainEnum now manages its own connection or receives one depending on implementation
		// For the standalone deamon, we let it function as is, or update it to take db
		if err := SubdomainEnum(url, rateLimit); err != nil {
			localUtils.CheckError(err)
		}
	}
}

func CheckTables() {
	db, err := OpenDatabase()
	if err != nil {
		localUtils.CheckError(err)
		return
	}
	defer db.Close()

	// --- Enable Foreign Keys for SQLite ---
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		localUtils.CheckError(err)
		return
	}

	// --- 1. Existing Result Tables ---
	if err := createTargetTableIfNotExists(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createSubsTableIfNotExists(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createUrlsTableIfNotExist(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createSpiderTableIfNotExist(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createTargetPatternsIfNotExists(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createVulnerabilitiesTable(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createFuzzingResultsTable(db); err != nil {
		localUtils.CheckError(err)
		return
	}

	// --- 2. New Workflow Tables ---
	if err := createProcFuncsTable(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createProcPathsTable(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createBranchingRulesTable(db); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createProcPathItemsTable(db); err != nil {
		localUtils.CheckError(err)
		return
	}

	// --- 3. Seed Default Data ---
	if err := SeedDefaultWorkflow(db); err != nil {
		localUtils.CheckError(err)
		return
	}

	localUtils.Logger("Database tables and seed data checked successfully", 1)
}
