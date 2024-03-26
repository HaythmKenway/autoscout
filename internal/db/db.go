package db

import (
	"database/sql"
	"fmt"
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
		fmt.Println(err)
		return
	}
	defer db.Close()

	if err := createTargetTableIfNotExists(); err != nil {
		fmt.Println(err)
		return
	}

	urls, err := GetTargetsFromTable()
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, url := range urls {
		if err := SubdomainEnum(url); err != nil {
			fmt.Println(err)
		}
	}
}

func openDatabase() (*sql.DB, error) {
	return sql.Open("sqlite3", DatabaseFile)
}
