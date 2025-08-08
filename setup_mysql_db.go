package main

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var DB *gorm.DB

func SetupDB() {
	db, err := gorm.Open(mysql.Open(MYSQL_URI), &gorm.Config{})
	if err != nil {
		panic(err.Error())
	}

	DB = db
}
