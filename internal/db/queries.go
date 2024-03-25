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
func createUrlsTableIfNotExists(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS urls (
			subdomain TEXT,
			url TEXT PRIMARY KEY,
			statusCode INTEGER,
			ipAddress TEXT,
			lastModified DATE DEFAULT CURRENT_TIMESTAMP)`)
	return err
}
