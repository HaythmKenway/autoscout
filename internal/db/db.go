package db

import (
	"database/sql"
	"fmt"
	"os/exec"

	"github.com/HaythmKenway/autoscout/pkg/utils"
	_ "github.com/mattn/go-sqlite3"
)

type Configuration struct {
	DatabaseFile string
}

func ClearDB() error {
	cmd := exec.Command("rm", utils.GetWorkingDirectory()+"/autoscout.db")
	return cmd.Run()
}

func Deamon() {
	config := Configuration{
		DatabaseFile: utils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer db.Close()

	if err := createTableIfNotExists(db, "targets"); err != nil {
		fmt.Println(err)
		return
	}

	urls, err := getTargetsFromTable(db)
	if err != nil {
		fmt.Println(err)
		return
	}

	for _, url := range urls {
		if err := SubdomainEnum(config, url, db); err != nil {
			fmt.Println(err)
		}
	}
}

func openDatabase(filename string) (*sql.DB, error) {
	return sql.Open("sqlite3", filename)
}

