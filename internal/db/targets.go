package db

import (
	"database/sql"
	"fmt"
	URL "net/url"
	"time"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func AddTarget(url string) (string, error) { 
	u, err := URL.ParseRequestURI("http://" + url)
	url = u.Hostname()
	if err != nil {
		return "Invalid Domain", err
	}
	db, err := openDatabase()
	if err != nil {
		return "Error opening Database", err
	}
	defer db.Close()

	createTargetTableIfNotExists()
	_, err = db.Exec("INSERT INTO targets (subdomain) VALUES (?)", url)
	if err != nil {
		return "Error inserting into Database", err
	}
	return "Target added successfully", nil
}
func RemoveTarget(url string) (string, error) {
	db, err := openDatabase()
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

func ScanCompleted(target string) {
	db, err := openDatabase()
	localUtils.CheckError(err)
	defer db.Close()

	stmt, err := db.Prepare("UPDATE targets SET lastScanned = $1 WHERE subdomain = $2")
	localUtils.CheckError(err)
	defer stmt.Close()

	_, err = stmt.Exec(time.Now(), target)
	localUtils.CheckError(err)
}


func GetTargetsFromTable(daysOpt ...int) ([]string, error) {
	days := 0
	if len(daysOpt) > 0 {
		days = daysOpt[0]}

	db, err := openDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	var (
		stmt *sql.Stmt
		rows *sql.Rows
	)

	if days == 0 {
		stmt, err = db.Prepare("SELECT subdomain FROM targets")
		if err != nil {
			return nil, err
		}
		defer stmt.Close()

		rows, err = stmt.Query()
		if err != nil {
			return nil, err
		}
	} else {
		stmt, err = db.Prepare(`SELECT subdomain FROM targets WHERE lastScanned < $1 OR lastScanned is NULL`)
		if err != nil {
			return nil, fmt.Errorf("prepare error: %w", err)
		}
		defer stmt.Close()

		cutoff := time.Now().AddDate(0, 0, -days)
		rows, err = stmt.Query(cutoff)
		if err != nil {
			return nil, fmt.Errorf("query error: %w", err)
		}
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

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return urls, nil
}
