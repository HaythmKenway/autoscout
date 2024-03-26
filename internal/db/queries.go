package db

import (
	"database/sql"
	"fmt"
)

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
			domain TEXT,
			subdomain TEXT PRIMARY KEY,
			lastModified DATE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}

func createUrlsTableIfNotExist(db *sql.DB) error{
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			domain TEXT,
			title TEXT,
			url TEXT PRIMARY KEY,
			host TEXT,
			scheme TEXT,
			a TEXT,
			cname TEXT,
			tech TEXT,
			ip TEXT,
			port TEXT,
			status_code TEXT,
			lastModified DATE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}
