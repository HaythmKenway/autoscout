package db

// job done
func createTargetTableIfNotExists() error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS targets (
            lastModified DATE DEFAULT CURRENT_TIMESTAMP,
            subdomain TEXT PRIMARY KEY
        )
    `)
	return err
}

func createSubsTableIfNotExists() error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS subdomain (
            domain TEXT,
            subdomain TEXT PRIMARY KEY,
            lastModified DATE DEFAULT CURRENT_TIMESTAMP
        )
    `)
	return err
}

func createUrlsTableIfNotExist() error {
	db, err := openDatabase()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
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
