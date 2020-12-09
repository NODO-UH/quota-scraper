package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/NODO-UH/quota-scraper/src/scraper"
)

var logErr *log.Logger
var logInfo *log.Logger

func init() {
	logErr = log.New(os.Stderr, "ERROR [main]: ", 1)
	logInfo = log.New(os.Stdout, "INFO [main]: ", 1)
}

func main() {
	squidFile := flag.String("file", "squid.logs", "Path to squid file with logs")
	dbUri := flag.String("db-uri", "", "MongoDB Connection URI")
	outputLog := flag.String("out-log", "quota-scrapper.logs", "Path to file to program logs")
	cores := flag.Int("cores", runtime.NumCPU(), "Max number of cores")
	flag.Parse()
	logFile, err := os.OpenFile(*outputLog, os.O_CREATE|os.O_WRONLY,0644)
	if err != nil {
		logErr.Printf("couldn't open log file")
	}
	logErr.SetOutput(logFile)
	logInfo.SetOutput(logFile)

	logInfo.Printf("setting up with %d cores", *cores)
	runtime.GOMAXPROCS(*cores)

	logInfo.Println(fmt.Sprintf("squid file: %s", *squidFile))

	if *dbUri == "" {
		logErr.Fatal("mongodb connection uri is missing")
	} else {
		go database.StartDatabase(*dbUri, logFile)
	}

	<-database.UpOk

	alreadyOpenError := false
	var lastDateTime float64 = database.GetLastDateTime()
	scraper.SetLogOutput(logFile)
	for {
		file, err := os.Open(*squidFile)
		if err != nil {
			if !alreadyOpenError {
				logErr.Println(err)
				alreadyOpenError = true
			}
		} else {
			logInfo.Println(fmt.Sprintf("parsing file %s", file.Name()))
			alreadyOpenError = false
			err, lastDateTime = scraper.ParseFile(file, lastDateTime)
			if err != nil {
				logErr.Println(err)
			}
		}
		time.Sleep(3 * time.Second)
	}
}
