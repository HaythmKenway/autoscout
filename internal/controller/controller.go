package controller

import (
	"github.com/HaythmKenway/autoscout/internal/db"
	"github.com/HaythmKenway/autoscout/pkg/spider"
	"github.com/HaythmKenway/autoscout/pkg/localUtils"

)
func Init() {

	db.CheckTables()
}

func Spider(domain string) {
	targets,err:=spider.Spider(domain)
	localUtils.CheckError(err)
	db.AddSpiderTargets(domain,targets)
}

