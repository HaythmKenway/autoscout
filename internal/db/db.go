package db 

import (
	"database/sql"
	"fmt"
	"os/exec"
	_ "github.com/mattn/go-sqlite3"
	"log"
	"github.com/HaythmKenway/autoscout/pkg/utils"
	"github.com/HaythmKenway/autoscout/pkg/subdomain"
	"github.com/HaythmKenway/autoscout/pkg/notifier"
)

type Configuration struct {
	DatabaseFile string
}

func ClearDB() {
	cmd := exec.Command("rm", utils.GetWorkingDirectory() + "/autoscout.db")
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}
func Cron() {
	config := Configuration{
		DatabaseFile:utils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()
	err = createTableIfNotExists(db, "targets")
	fmt.Println("Creating targets table")
	if err != nil {
		fmt.Printf("Error creating targets table: %v\n", err)
		return
	}

	
	urls, err := getTargetsFromTable(db)
	if err != nil {
		fmt.Printf("Error getting URLs from the database: %v\n", err)
		return
	}
	log.Println("Starting SubdomainEnum")
	for _, url := range urls {
		err := SubdomainEnum(config, url, db)
		if err != nil {
			fmt.Printf("Error in SubdomainEnum for %s: %v\n", url, err)
		}
	}
}

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
func openDatabase(filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	return db, nil
}

func SubdomainEnum(config Configuration, url string, db *sql.DB) error {
	err := createSubsTableIfNotExists(db)
	if err != nil {
		log.Printf("Error creating subdomain table: %v\n", err)
		return err
	}

	prev, err := getSubsFromTable(db,url)
	if err != nil {
		log.Printf("Error getting subdomains from the database: %v\n", err)
		return err
	}

	now, err := subdomain.Subdomain(url)
	if err != nil {
		log.Printf("Error getting subdomains  2from the database: %v\n", err)
		return err
	}

	insertElement := utils.ElementsOnlyInNow(prev, now)
	notifier.ClassifyNotification(insertElement)
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Error beginning transaction: %v\n", err)
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

func createTableIfNotExists(db *sql.DB, tableName string) error {
	_, err := db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			lastModified DATE DEFAULT CURRENT_TIMESTAMP,
			subdomain TEXT PRIMARY KEY
		)
	`, tableName))
	return err
}
func createSubsTableIfNotExists(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS subdomain (
			lastModified DATE DEFAULT CURRENT_TIMESTAMP,
			subdomain TEXT PRIMARY KEY,
			domain TEXT
		)
	`)
	return err
}


