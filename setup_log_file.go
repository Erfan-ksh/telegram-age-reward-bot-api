package main

import (
	"log"
	"os"
)

var ErrorLogger *log.Logger
var InfoLogger *log.Logger

func SetupLogFile() {
	exists, err := FolderExists("./logs")
	if err != nil {
		panic("could not check logs folder")
	}

	var f, f2 *os.File
	if !exists {
		err = os.Mkdir("./logs", os.ModePerm)
		if err != nil {
			panic("could not create logs folder path: ./logs")
		}

		f, err = os.Create("./logs/errors.log")
		if err != nil {
			panic("could not create errors.log file path: ./logs")
		}

		f2, err = os.Create("./logs/info.log")
		if err != nil {
			panic("could not create errors.log file path: ./logs")
		}
	} else {
		f, err = os.OpenFile("./logs/errors.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			panic("error opening file")
		}

		f2, err = os.OpenFile("./logs/info.log", os.O_APPEND|os.O_CREATE|os.O_RDWR, 0666)
		if err != nil {
			panic("error opening file")
		}
	}

	ErrorLogger = log.New(f, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
	InfoLogger = log.New(f2, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}
