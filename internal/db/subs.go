package db

import (
	"database/sql"
	"fmt"
	"log"
)

func getSubsFromTable(db *sql.DB, domain string) ([]string, error) {
	selectStmt, err := db.Prepare("SELECT subdomain FROM subdomain WHERE domain = ?")
	if err != nil {
		log.Printf("Error preparing select statement: %v\n", err)
		return nil, err
	}
	defer selectStmt.Close()

	rows, err := selectStmt.Query(domain)
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

func checkErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}

func AddSubs(db *sql.DB, url string, domain string) error {
	_, err := db.Exec("INSERT INTO subdomain (subdomain,domain) VALUES (?,?)", url, domain)
	return err
}
