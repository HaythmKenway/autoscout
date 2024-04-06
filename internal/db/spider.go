package db

import (
	"github.com/HaythmKenway/autoscout/pkg/localUtils"
)
func AddSpiderTargets(domain string, targets []string) {
	db, err := openDatabase()
	localUtils.CheckError(err)
	defer db.Close()
	
	stmt, err := db.Prepare("INSERT INTO spider(domain,url) VALUES(?,?)")
	localUtils.CheckError(err)
	
	for _, target := range targets {
		if target != ""{ 
		_, err = stmt.Exec(domain, target)
		localUtils.CheckError(err)
	}}
	defer stmt.Close()
	localUtils.Logger("Targets added for the domain "+domain, 1)

}
