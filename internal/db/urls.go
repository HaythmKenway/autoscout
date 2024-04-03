package db

import (
	"fmt"

	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)

func AddUrl(title string, url string, host string, scheme string, a string, cname string, tech string, ip string, port string, status_code string) {
	db, err := openDatabase()
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error opening database: %v\n", err), 2)
		return
	}
	defer db.Close()
	if err := createUrlsTableIfNotExist(); err != nil {
		localUtils.Logger(fmt.Sprintf("Error creating table: %v\n", err), 2)
		return
	}
	stmt, err := db.Prepare("INSERT INTO urls(title,url,host,scheme,a,cname,tech,ip,port,status_code) VALUES(?,?,?,?,?,?,?,?,?,?) ON CONFLICT (url) DO UPDATE SET title=excluded.title, host=excluded.host, scheme=excluded.scheme, a=excluded.a, cname=excluded.cname, tech=excluded.tech, ip=excluded.ip, port=excluded.port, status_code=excluded.status_code")
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error preparing statement: %v\n", err), 2)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(title, url, host, scheme, a, cname, tech, ip, port, status_code)
	if err != nil {
		localUtils.Logger(fmt.Sprintf("Error inserting data: %v\n", err), 2)
		return
	}
	localUtils.Logger("Data inserted successfully", 1)

}

func GetDataFromTable(Tgturl string) ([]string, error) {
	db, err := openDatabase()
	localUtils.CheckError(err)
	defer db.Close()

	rows, err := db.Query("SELECT * FROM urls WHERE url LIKE ?", "%"+Tgturl+"%")
	localUtils.CheckError(err)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var title string
	var url string
	var host string
	var scheme string
	var a string
	var cname string
	var tech string
	var ip string
	var port string
	var status_code string
	var lastModified string

	for rows.Next() {
		err = rows.Scan(&title, &url, &host, &scheme, &a, &cname, &tech, &ip, &port, &status_code, &lastModified)
		localUtils.CheckError(err)
	}
	localUtils.CheckError(err)
	if url == "" {
		return nil, fmt.Errorf("target not found")
	}
	var res = []string{title, url, host, scheme, a, cname, tech, ip, port, status_code}
	return res, err
}
