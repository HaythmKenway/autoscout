package db

import (
	"database/sql"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/notifier"
	"github.com/HaythmKenway/autoscout/pkg/subdomain"
	"github.com/charmbracelet/log"
)

func SubdomainEnum(target string) error {
	// 1. Controller manages connection
	db, err := OpenDatabase()
	if err != nil {
		localUtils.CheckError(err)
		return err
	}
	defer db.Close()

	// 2. Pass DB to getter
	prev, err := GetSubsFromTable(db, target)
	if err != nil {
		localUtils.CheckError(err)
		// continue even if error to try and get new ones
	}

	now, err := subdomain.Subdomain(target)
	localUtils.CheckError(err)

	localUtils.Logger(fmt.Sprintf("Previous subdomains count: %d", len(prev)), 3)
	insertElement := localUtils.ElementsOnlyInNow(prev, now)
	localUtils.Logger(fmt.Sprintf("New subdomains found: %d", len(insertElement)), 3)

	notifier.ClassifyNotification(insertElement)

	// 3. Start Transaction
	tx, err := db.Begin()
	localUtils.CheckError(err)

	for _, subd := range insertElement {
		err = AddSubs(tx, subd, target)
		if err != nil {
			log.Errorf("Error adding subdomain to database: %v", err)
			tx.Rollback()
			return err
		}
		fmt.Println(subd)
	}

	err = tx.Commit()
	localUtils.CheckError(err)
	return nil
}

func GetSubsFromTable(db *sql.DB, domain string) ([]string, error) {
	localUtils.Logger(fmt.Sprintf("Getting subdomains for domain: %v", domain), 1)

	selectStmt, err := db.Prepare("SELECT subdomain FROM subdomain WHERE domain = ?")
	if err != nil {
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
	return urls, nil
}

// AddSubs now accepts a Transaction, not the whole DB
func AddSubs(tx *sql.Tx, url string, domain string) error {
	_, err := tx.Exec("INSERT INTO subdomain (subdomain,domain) VALUES (?,?)", url, domain)
	return err
}
