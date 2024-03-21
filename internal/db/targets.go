package db

import (
	"database/sql"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/utils"
)

func AddTarget(url string) {
	config := Configuration{
		DatabaseFile: utils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	createTableIfNotExists(db, "targets")
	_, err = db.Exec("INSERT INTO targets (subdomain) VALUES (?)", url)
	if err != nil {
		fmt.Printf("Error inserting target: %v\n", err)
		return
	}
}
func getTargetsFromTable(db *sql.DB) ([]string, error) {
	selectStmt, err := db.Prepare("SELECT subdomain FROM targets")
	if err != nil {
		return nil, err
	}
	defer selectStmt.Close()

	rows, err := selectStmt.Query()
	if err != nil {
		return nil, err
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
