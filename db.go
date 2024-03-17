package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"os/exec"
	"log"
)

type Configuration struct {
	DatabaseFile string
}

func cron() {
	config := Configuration{
		DatabaseFile: getWorkingDirectory() + "/autoscout.db",
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

	
	urls, err := getUrlsFromTable(db, "targets")
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

func addTarget(url string) {
	config := Configuration{
		DatabaseFile: getWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		fmt.Printf("Error opening database: %v\n", err)
		return
	}
	defer db.Close()

	_, err = db.Exec("INSERT INTO targets (url) VALUES (?)", url)
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
	tableName := removeSpecialCharacters(url)
	err := createTableIfNotExists(db, tableName)
	if err != nil {
		return err
	}

	prev, err := getUrlsFromTable(db, tableName)
	if err != nil {
		return err
	}

	now, err := subdomain(url)
	if err != nil {
		return err
	}

	insertElement := ElementsOnlyInNow(prev, now)

	pipeReader, pipeWriter := io.Pipe()
	cmd := exec.Command("notify", "-mf", "🎯 New Target Found! \n {{data}}" )
	cmd.Stdin = pipeReader
	done := make(chan error)

	go func() {
		// Start the command and capture any errors
		err := cmd.Run()
		done <- err
	}()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for _, u := range insertElement {
		_, err := pipeWriter.Write([]byte(u + "\n"))
		if err != nil {
			return err
		}

		_, err = tx.Exec(fmt.Sprintf("INSERT INTO %s (url) VALUES (?)", tableName), u)
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
			id INTEGER PRIMARY KEY,
			url TEXT
		)
	`, tableName))
	return err
}

func getUrlsFromTable(db *sql.DB, tableName string) ([]string, error) {
	selectStmt, err := db.Prepare(fmt.Sprintf("SELECT url FROM %s", tableName))
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

func checkErr(err error) {
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}
}


