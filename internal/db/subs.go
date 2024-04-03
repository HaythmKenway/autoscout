package db

import (
	"database/sql"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/notifier"
	"github.com/HaythmKenway/autoscout/pkg/subdomain"
	"github.com/charmbracelet/log"
)

func SubdomainEnum(url string) error { //*****controller//*****
	db, err := openDatabase()
	localUtils.CheckError(err)
	defer db.Close()

	err = createSubsTableIfNotExists()
	localUtils.CheckError(err)

	prev, err := GetSubsFromTable(url)
	localUtils.CheckError(err)

	now, err := subdomain.Subdomain(url)
	localUtils.CheckError(err)

	localUtils.Logger(fmt.Sprintf("Previous subdomains: %v\n", prev), 3)
	insertElement := localUtils.ElementsOnlyInNow(prev, now)
	localUtils.Logger(fmt.Sprintf("New subdomains: %v\n", insertElement), 3)
	notifier.ClassifyNotification(insertElement)
	tx, err := db.Begin()
	localUtils.CheckError(err)

	for _, subd := range insertElement {
		err = AddSubs(db, subd, url)
		if err != nil {
			log.Errorf("Error adding subdomain to database: %v\n", err)
			tx.Rollback()
			return err
		}
	}

	err = tx.Commit()
	localUtils.CheckError(err)
	for _, x := range insertElement {
		fmt.Println(x)
	}
	return nil
}

func GetSubsFromTable(domain string) ([]string, error) {
	localUtils.Logger(fmt.Sprintf("Getting subdomains for domain: %v\n", domain), 1)
	db, err := openDatabase()
	localUtils.CheckError(err)
	defer db.Close()
	createSubsTableIfNotExists()
	selectStmt, err := db.Prepare("SELECT subdomain FROM subdomain WHERE domain = ?")
	localUtils.CheckError(err)
	defer selectStmt.Close()

	rows, err := selectStmt.Query(domain)
	localUtils.CheckError(err)
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
func AddSubs(db *sql.DB, url string, domain string) error {
	_, err := db.Exec("INSERT INTO subdomain (subdomain,domain) VALUES (?,?)", url, domain)
	return err
}
