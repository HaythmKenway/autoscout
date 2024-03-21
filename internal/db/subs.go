package db

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/HaythmKenway/autoscout/pkg/notifier"
	"github.com/HaythmKenway/autoscout/pkg/subdomain"
	"github.com/HaythmKenway/autoscout/pkg/utils"
)

func SubdomainEnum(config Configuration, url string, db *sql.DB) error {
	err := createSubsTableIfNotExists(db)
	if err != nil {
		return err
	}

	prev, err := getSubsFromTable(db, url)
	if err != nil {
		return err
	}

	now, err := subdomain.Subdomain(url)
	if err != nil {
		return err
	}

	insertElement := utils.ElementsOnlyInNow(prev, now)
	notifier.ClassifyNotification(insertElement)
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, subd := range insertElement {
		err = AddSubs(db, subd, url)
		if err != nil {

			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	for _, x := range insertElement {
		fmt.Println(x)
	}
	return nil
}

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
