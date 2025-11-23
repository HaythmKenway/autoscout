package db

import (
	"database/sql"
	"fmt"
	URL "net/url"
	"strings"
	"time"
)

// AddTarget manages its own connection (CLI tool usage)
func AddTarget(input string) (string, error) {
	if !strings.HasPrefix(input, "http") {
		input = "http://" + input
	}
	u, err := URL.ParseRequestURI(input)
	if err != nil {
		return "Invalid Domain", err
	}
	hostname := u.Hostname()

	db, err := OpenDatabase()
	if err != nil {
		return "Error opening Database", err
	}
	defer db.Close()

	// Ensure table exists (safe redundancy for CLI)
	createTargetTableIfNotExists(db)

	_, err = db.Exec("INSERT OR IGNORE INTO targets (subdomain) VALUES (?)", hostname)
	if err != nil {
		return "Error inserting into Database", err
	}
	return "Target added successfully", nil
}

// RemoveTarget manages its own connection (CLI tool usage)
func RemoveTarget(url string) (string, error) {
	db, err := OpenDatabase()
	if err != nil {
		return "Error opening Database", err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM targets WHERE subdomain = ?", url)
	if err != nil {
		return "Error deleting from Database", err
	}
	return "Target removed successfully", nil
}

// ScanCompleted accepts DB connection (Used by Scheduler/Workers)
func ScanCompleted(db *sql.DB, target string) error {
	stmt, err := db.Prepare("UPDATE targets SET lastScanned = ? WHERE subdomain = ?")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.Exec(time.Now(), target)
	return err
}

// GetTargetsFromTable accepts DB connection (Used by Scheduler)
func GetTargetsFromTable(db *sql.DB, daysOpt ...int) ([]string, error) {
	days := 0
	if len(daysOpt) > 0 {
		days = daysOpt[0]
	}

	var (
		stmt *sql.Stmt
		err  error
		rows *sql.Rows
	)

	if days == 0 {
		stmt, err = db.Prepare("SELECT subdomain FROM targets")
		if err != nil {
			return nil, err
		}
	} else {
		stmt, err = db.Prepare(`SELECT subdomain FROM targets WHERE lastScanned < ? OR lastScanned is NULL`)
		if err != nil {
			return nil, fmt.Errorf("prepare error: %w", err)
		}
	}
	defer stmt.Close()

	if days == 0 {
		rows, err = stmt.Query()
	} else {
		cutoff := time.Now().AddDate(0, 0, -days)
		rows, err = stmt.Query(cutoff)
	}

	if err != nil {
		return nil, fmt.Errorf("query error: %w", err)
	}
	defer rows.Close()

	var urls []string
	for rows.Next() {
		var u string
		if err := rows.Scan(&u); err != nil {
			return nil, err
		}
		urls = append(urls, u)
	}

	return urls, nil
}
