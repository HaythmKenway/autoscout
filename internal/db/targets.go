package db

import (
	URL "net/url"
)

func AddTarget(url string) (string, error) { //*****controller//*****
	u, err := URL.ParseRequestURI("http://" + url)
	url = u.Hostname()
	if err != nil {
		return "Invalid Domain", err
	}
	db, err := openDatabase()
	if err != nil {
		return "Error opening Database", err
	}
	defer db.Close()

	createTargetTableIfNotExists()
	_, err = db.Exec("INSERT INTO targets (subdomain) VALUES (?)", url)
	if err != nil {
		return "Error inserting into Database", err
	}
	return "Target added successfully", nil
}
func RemoveTarget(url string) (string, error) { //*****controller//*****
	db, err := openDatabase()
	if err != nil {
		return "Error opening Database", err
	}
	defer db.Close()

	_, err = db.Exec("DELETE FROM targets WHERE subdomain = ?", url)
	if err != nil {
		return "Error deleting from Database", err
	}
	return "Target removed successfully", nil
}

func GetTargetsFromTable() ([]string, error) {
	db, err := openDatabase()
	if err != nil {
		return nil, err
	}
	defer db.Close()

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
