/*package main

import (
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"

	_ "github.com/mattn/go-sqlite3"
)
func cron(){
	fileName:=getWorkingDirectory()+"/autoscout.db"
	db,err := sql.Open("sqlite3",fileName)
	checkErr(err)
	defer db.Close()
	urls,_ :=getUrlsFromTable(db,"targets")
	for _,x:=range urls{
		checkErr(SubdomainEnum(x))
	}


}

func SubdomainEnum(url string) error {
	fileName := getWorkingDirectory() + "/autoscout.db"
	file, err := os.OpenFile(fileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	db, err := sql.Open("sqlite3", fileName)
	checkErr(err)
	defer db.Close()

	tableName := removeSpecialCharacters(url)
	_, err = db.Exec(fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id INTEGER PRIMARY KEY,
			url TEXT
		)
	`, tableName))
	checkErr(err)

	prev, err := getUrlsFromTable(db, tableName)
	checkErr(err)
	now, err := subdomain(url)
	checkErr(err)
	insertElement := ElementsOnlyInNow(prev, now)
	pipeReader, pipeWriter := io.Pipe()
	cmd := exec.Command("notify", "-mf", "🎯 New Target Found! \n {{data}}","-bulk")
	cmd.Stdin = pipeReader
	done := make(chan error)

	go func() {
		// Start the command and capture any errors
		err := cmd.Run()
		done <- err
	}()

	for _, u := range insertElement {
		_, err := pipeWriter.Write([]byte(u + "\n"))
		if err != nil {
			fmt.Println("Error writing to pipe:", err)
			break
		}
		_, err = db.Exec(fmt.Sprintf("INSERT INTO %s (url) VALUES (?)", tableName), u)
		checkErr(err)
	}

	for _, x := range insertElement {
		fmt.Println(x)
	}
	return nil // Return any relevant error, not nil
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
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
}*/

package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"io"
	"os/exec"
	"time"
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

	urls, err := getUrlsFromTable(db, "targets")
	if err != nil {
		fmt.Printf("Error getting URLs from the database: %v\n", err)
		return
	}

	for _, url := range urls {
		err := SubdomainEnum(config, url, db)
		if err != nil {
			fmt.Printf("Error in SubdomainEnum for %s: %v\n", url, err)
		}
	}
	time.Sleep(time.Hour/2)
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


