package db

import (
	"database/sql"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	"github.com/HaythmKenway/autoscout/pkg/notifier"
	"github.com/HaythmKenway/autoscout/pkg/subdomain"
	"github.com/charmbracelet/log"
)

func GetSubs(domain string) ([]string, error) {
	config := Configuration{
		DatabaseFile: localUtils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		log.Errorf("Error opening database: %v\n", err)
		return nil, err
	}
	defer db.Close()

	createSubsTableIfNotExists(db)
	return getSubsFromTable(db, domain)
}
func SubdomainFuzz(domain string) ([]string, error) {
	config := Configuration{
		DatabaseFile: localUtils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		log.Errorf("Error opening database: %v\n", err)
		return nil, err
	}
	SubdomainEnum(config, domain, db)
	defer db.Close()
	return getSubsFromTable(db, domain)
}

func SubdomainEnum(config Configuration, url string, db *sql.DB) error {
	err := createSubsTableIfNotExists(db)
	if err != nil {
		log.Errorf("Error creating subdomain table: %v\n", err)
		return err
	}

	prev, err := getSubsFromTable(db, url)
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
