package main

import (
	"log"
	"os"
)

// logger is a general log
var logger *log.Logger

func InitLogger() *log.Logger {
	writer := os.Stdout
	tmpLogger := log.New(writer, "", log.Ldate|log.Ltime|log.Lshortfile)

	return tmpLogger
}
