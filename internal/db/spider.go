package db

import (
	"database/sql"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

// AddSpiderTargets now accepts an existing DB connection
func AddSpiderTargets(db *sql.DB, domain string, targets []string) error {
	stmt, err := db.Prepare("INSERT OR IGNORE INTO spider(target,url) VALUES(?,?)")
	if err != nil {
		localUtils.CheckError(err)
		return err
	}
	defer stmt.Close()

	for _, target := range targets {
		if target != "" {
			_, err = stmt.Exec(domain, target)
			if err != nil {
				localUtils.CheckError(err)
			}
		}
	}
	localUtils.Logger("Spider targets added for domain "+domain, 1)
	return nil
}
