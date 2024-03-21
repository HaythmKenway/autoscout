package db

import (
	"database/sql"
	"fmt"
	"log"
	"os/exec"

	"github.com/HaythmKenway/autoscout/pkg/notifier"
	"github.com/HaythmKenway/autoscout/pkg/subdomain"
	"github.com/HaythmKenway/autoscout/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
)

type Configuration struct {
	DatabaseFile string
}

func ClearDB() {
	cmd := exec.Command("rm", utils.GetWorkingDirectory()+"/autoscout.db")
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}
func Cron() {
	config := Configuration{
		DatabaseFile: utils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		return
	}
	defer db.Close()
	err = createTableIfNotExists(db, "targets")
	fmt.Println("Creating targets table")
	if err != nil {
		return
	}

	urls, err := getTargetsFromTable(db)
	if err != nil {
		return
	}
	log.Println("Starting SubdomainEnum")
	for _, url := range urls {
		err := SubdomainEnum(config, url, db)
		if err != nil {
		}
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
