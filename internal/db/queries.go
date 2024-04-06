package db

import "github.com/HaythmKenway/autoscout/pkg/localUtils"

// job done
func createTargetTableIfNotExists() error {
	db, err := openDatabase()
	localUtils.CheckError(err)
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
	localUtils.CheckError(err)
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
	localUtils.CheckError(err)
	defer db.Close()

	_, err = db.Exec(`
        CREATE TABLE IF NOT EXISTS urls (
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
	return err}

func createSpiderTableIfNotExist() error {
	db, err := openDatabase()
	localUtils.CheckError(err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS spider (
			domain TEXT,
			url TEXT PRIMARY KEY,
			lastModified DATE DEFAULT CURRENT_TIMESTAMP
		)
	`)
	return err
}
