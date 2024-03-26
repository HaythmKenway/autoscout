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
	if err != nil {
		log.Printf("Error opening database: %v\n", err)
		return err
	}
	defer db.Close()

	err = createSubsTableIfNotExists()
	if err != nil {
		log.Errorf("Error creating subdomain table: %v\n", err)
		return err
	}

	prev, err := GetSubsFromTable(url)
	if err != nil {
		log.Errorf("Error getting subdomains from table: %v\n", err)
		return err
	}

	now, err := subdomain.Subdomain(url)
	if err != nil {
		log.Errorf("Error getting subdomains: %v\n", err)
		return err
	}
	log.Infof("Previous subdomains: %v\n", prev)
	insertElement := localUtils.ElementsOnlyInNow(prev, now)
	log.Infof("New subdomains: %v\n", insertElement)
	notifier.ClassifyNotification(insertElement)
	tx, err := db.Begin()
	if err != nil {
		log.Errorf("Error beginning transaction: %v\n", err)
		return err
	}

	for _, subd := range insertElement {
		err = AddSubs(db, subd, url)
		if err != nil {
			log.Errorf("Error adding subdomain to database: %v\n", err)
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

func GetSubsFromTable(domain string) ([]string, error) {
	db, err := openDatabase()
	if err != nil {
		log.Printf("Error opening database: %v\n", err)
		return nil, err
	}
	defer db.Close()

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
func AddSubs(db *sql.DB, url string, domain string) error {
	_, err := db.Exec("INSERT INTO subdomain (subdomain,domain) VALUES (?,?)", url, domain)
	return err
}
