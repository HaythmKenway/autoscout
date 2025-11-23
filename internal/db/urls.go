package db

import (
	"database/sql"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

// AddUrl inserts or updates a URL record.
// Accepts *sql.DB to reuse the connection from the worker/scheduler.
func AddUrl(db *sql.DB, title string, url string, host string, scheme string, a string, cname string, tech string, ip string, port string, status_code string) error {
	query := `INSERT INTO urls(title,url,host,scheme,a,cname,tech,ip,port,status_code) 
              VALUES(?,?,?,?,?,?,?,?,?,?) 
              ON CONFLICT (url) DO UPDATE SET 
              title=excluded.title, host=excluded.host, scheme=excluded.scheme, 
              a=excluded.a, cname=excluded.cname, tech=excluded.tech, 
              ip=excluded.ip, port=excluded.port, status_code=excluded.status_code`

	_, err := db.Exec(query, title, url, host, scheme, a, cname, tech, ip, port, status_code)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error inserting URL data: %v", err), 2)
		return err
	}

	localUtils.Logger("URL Data inserted/updated successfully", 1)
	return nil
}

// GetDataFromTable searches for a URL and returns its details.
// Accepts *sql.DB to reuse the connection.
func GetDataFromTable(db *sql.DB, Tgturl string) ([]string, error) {
	// Use the passed DB connection
	rows, err := db.Query("SELECT * FROM urls WHERE url LIKE ?", "%"+Tgturl+"%")
	if err != nil {
		localUtils.CheckError(err)
		return nil, err
	}
	defer rows.Close()

	var (
		title        string
		url          string
		host         string
		scheme       string
		a            string
		cname        string
		tech         string
		ip           string
		port         string
		status_code  string
		lastModified string
		found        bool
	)

	// Iterate through rows.
	// NOTE: Since this returns a single []string, it effectively returns the LAST match found.
	for rows.Next() {
		err = rows.Scan(&title, &url, &host, &scheme, &a, &cname, &tech, &ip, &port, &status_code, &lastModified)
		if err != nil {
			localUtils.CheckError(err)
			continue
		}
		found = true
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	if !found {
		return nil, fmt.Errorf("target not found")
	}

	// Return the columns as a slice of strings
	return []string{title, url, host, scheme, a, cname, tech, ip, port, status_code}, nil
}
