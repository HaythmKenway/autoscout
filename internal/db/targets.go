package db

import (
	"database/sql"
)
func getTargetsFromTable(db *sql.DB) ([]string, error) {
	selectStmt, err := db.Prepare("SELECT subdomain FROM targets")
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
