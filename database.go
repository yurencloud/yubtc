package main

import (
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/sqlite"
	log "github.com/sirupsen/logrus"
)

var db *gorm.DB

func init() {
	db, err := gorm.Open("sqlite3", "./blockchain.db")
	if err != nil {
		log.Panic(err)
	}
	defer db.Close()
}

func Db() *gorm.DB {
	return db
}