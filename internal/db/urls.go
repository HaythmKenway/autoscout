package db

import (
	"fmt"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"

)

func AddUrl(title string ,url string ,host string ,scheme string ,a string ,cname string ,tech string ,ip string,port string, status_code string){
	config := Configuration{
		DatabaseFile: localUtils.GetWorkingDirectory() + "/autoscout.db",
	}
	db, err := openDatabase(config.DatabaseFile)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error opening database: %v\n", err), 2)
		return 
	}
	defer db.Close()
	if err := createUrlsTableIfNotExist(db); err != nil {
		localUtils.Logger(fmt.Sprintf("Error creating table: %v\n", err), 2)
		return 
	}
	stmt, err := db.Prepare("INSERT INTO urls(title,url,host,scheme,a,cname,tech,ip,port,status_code) values(?,?,?,?,?,?,?,?,?,?)")
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error preparing statement: %v\n", err), 2)
		return 
	}
	defer stmt.Close()
	
	_, err = stmt.Exec(title,url,host,scheme,a,cname,tech,ip,port,status_code)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error inserting data: %v\n", err), 2)
		return 
	}
	localUtils.Logger("Data inserted successfully", 1)

}
