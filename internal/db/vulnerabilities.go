package db

import (
	"database/sql"
)

func createVulnerabilitiesTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS vulnerabilities (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target TEXT,
		type TEXT,
		proof TEXT,
		tool_source TEXT,
		severity TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(query)
	return err
}

func createFuzzingResultsTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS fuzzing_results (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		target TEXT,
		found_path TEXT,
		parameter TEXT,
		tool_source TEXT,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := db.Exec(query)
	return err
}

func AddVulnerability(db *sql.DB, target, vType, proof, source, severity string) error {
	query := `INSERT INTO vulnerabilities (target, type, proof, tool_source, severity) VALUES (?, ?, ?, ?, ?)`
	_, err := db.Exec(query, target, vType, proof, source, severity)
	return err
}

func AddFuzzResult(db *sql.DB, target, path, param, source string) error {
	query := `INSERT INTO fuzzing_results (target, found_path, parameter, tool_source) VALUES (?, ?, ?, ?)`
	_, err := db.Exec(query, target, path, param, source)
	return err
}

func GetVulnerabilities(db *sql.DB) ([]string, error) {
	query := `SELECT target || " | " || type || " (" || severity || ")" FROM vulnerabilities ORDER BY timestamp DESC`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []string
	for rows.Next() {
		var res string
		if err := rows.Scan(&res); err == nil {
			results = append(results, res)
		}
	}
	return results, nil
}
