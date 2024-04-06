package db

import (
	"database/sql"
	"os/exec"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
	_ "github.com/mattn/go-sqlite3"
)

var DatabaseFile = localUtils.GetWorkingDirectory() + "/autoscout.db"

func ClearDB() error {
	cmd := exec.Command("rm", DatabaseFile)
	return cmd.Run()
}

func Deamon() {
	db, err := openDatabase()
	if err != nil {
		localUtils.CheckError(err)
		return
	}
	defer db.Close()

	if err := createTargetTableIfNotExists(); err != nil {
		localUtils.CheckError(err)
		return
	}

	urls, err := GetTargetsFromTable()
	if err != nil {
		localUtils.CheckError(err)
		return
	}

	for _, url := range urls {
		if err := SubdomainEnum(url); err != nil {
			localUtils.CheckError(err)
		}
	}
}

func CheckTables() {
	db, err := openDatabase()
	if err != nil {
		localUtils.CheckError(err)
		return
	}
	defer db.Close()

	if err := createTargetTableIfNotExists(); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createSubsTableIfNotExists(); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createUrlsTableIfNotExist(); err != nil {
		localUtils.CheckError(err)
		return
	}
	if err := createSpiderTableIfNotExist(); err != nil {
		localUtils.CheckError(err)
		return
	}
}
func openDatabase() (*sql.DB, error) {
	return sql.Open("sqlite3", DatabaseFile)
}
