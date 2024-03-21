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

func ClearDB() {
	cmd := exec.Command("rm", utils.GetWorkingDirectory()+"/autoscout.db")
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}
func Deamon() {
	config := Configuration{
		DatabaseFile: utils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		return
	}
	defer db.Close()
	err = createTableIfNotExists(db, "targets")
	if err != nil {
		return
	}

	urls, err := getTargetsFromTable(db)
	if err != nil {
		return
	}
	for _, url := range urls {
		err := SubdomainEnum(config, url, db)
		if err != nil {
		}
	}
}
func openDatabase(filename string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil, err
	}
	return db, nil
}


