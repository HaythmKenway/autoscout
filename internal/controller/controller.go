package controller

import (
	"github.com/HaythmKenway/autoscout/internal/db"
)
func Init() {

	db.CheckTables()
}


