package main

import (
	"github.com/joho/godotenv"
	"os"
)

var MYSQL_URI string
var BOT_TOKEN string

func SetupEnvs() {
	err := godotenv.Load("./.env")
	if err != nil {
		panic("could not load env file")
	}

	MYSQL_URI = os.Getenv("MYSQL_URI")
	BOT_TOKEN = os.Getenv("BOT_TOKEN")
}
