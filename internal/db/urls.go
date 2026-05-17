package db

import (
	"database/sql"
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

// AddUrl inserts or updates a URL record.
// Accepts *sql.DB to reuse the connection from the worker/scheduler.
func AddUrl(db *sql.DB, subdomain string, title string, url string, host string, scheme string, a string, cname string, tech string, ip string, port string, status_code string, sessionID string) error {
	query := `INSERT INTO urls(subdomain, title, url, host, scheme, a, cname, tech, ip, port, status_code, session_id) 
              VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?) 
              ON CONFLICT (url) DO UPDATE SET 
              subdomain = excluded.subdomain,
              title = CASE WHEN excluded.title != '' THEN excluded.title ELSE urls.title END,
              host = excluded.host,
              scheme = CASE WHEN excluded.scheme != '' THEN excluded.scheme ELSE urls.scheme END,
              a = CASE WHEN excluded.a != '' THEN excluded.a ELSE urls.a END,
              cname = CASE WHEN excluded.cname != '' THEN excluded.cname ELSE urls.cname END,
              tech = CASE WHEN excluded.tech != '' THEN excluded.tech ELSE urls.tech END,
              ip = CASE WHEN excluded.ip != '' THEN excluded.ip ELSE urls.ip END,
              port = CASE WHEN excluded.port != '' THEN excluded.port ELSE urls.port END,
              status_code = CASE WHEN excluded.status_code != '' THEN excluded.status_code ELSE urls.status_code END,
              session_id = CASE WHEN excluded.session_id != '' THEN excluded.session_id ELSE urls.session_id END,
              lastModified = CURRENT_TIMESTAMP`

	_, err := db.Exec(query, subdomain, title, url, host, scheme, a, cname, tech, ip, port, status_code, sessionID)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error inserting URL data: %v", err), 2)
		return err
	}
	localUtils.Logger("URL Data inserted/updated successfully", 3)
	return nil
}

// GetUrlsBySubdomain returns all URLs associated with a specific subdomain or parent target.
func GetUrlsBySubdomain(db *sql.DB, sub string) ([]map[string]string, error) {
	query := "SELECT url, status_code, tech, session_id FROM urls WHERE subdomain = ? OR host = ?"
	rows, err := db.Query(query, sub, sub)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]string
	for rows.Next() {
		var u, status, tech, sessionID string
		if err := rows.Scan(&u, &status, &tech, &sessionID); err == nil {
			results = append(results, map[string]string{
				"url":        u,
				"status":     status,
				"tech":       tech,
				"session_id": sessionID,
			})
		}
	}
	return results, nil
}

// GetDataFromTable searches for a URL and returns its details.
// Accepts *sql.DB to reuse the connection.
func GetDataFromTable(db *sql.DB, Tgturl string) ([]string, error) {
	// Use the passed DB connection
	query := "SELECT subdomain, title, url, host, scheme, a, cname, tech, ip, port, status_code, lastModified FROM urls WHERE url LIKE ?"
	rows, err := db.Query(query, "%"+Tgturl+"%")
	if err != nil {
		localUtils.CheckError(err)
		return nil, err
	}
	defer rows.Close()

	var (
		subdomain    string
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
	for rows.Next() {
		err = rows.Scan(&subdomain, &title, &url, &host, &scheme, &a, &cname, &tech, &ip, &port, &status_code, &lastModified)
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
