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
	squid_file := flag.String("file", "squid.logs", "Path to squid file with logs")
	db_uri := flag.String("db-uri", "", "MongoDB Connection URI")
	cores := flag.Int("cores", runtime.NumCPU(), "max number of cores")
	logsPath := flag.String("logs", "squid-parser.logs", "path to file for logs")
	flag.Parse()

	if logsFile, err := os.OpenFile(*logsPath, os.O_CREATE|os.O_WRONLY, 0666); err != nil {
		logErr.Fatal(err)
	} else {
		logInfo.SetOutput(logsFile)
		logErr.SetOutput(logsFile)
		database.SetLogOutput(logsFile)
		scraper.SetLogOutput(logsFile)
	}

	logInfo.Printf("setting up with %d cores", *cores)
	runtime.GOMAXPROCS(*cores)

	logInfo.Println(fmt.Sprintf("squid file: %s", *squid_file))

	if *db_uri == "" {
		logErr.Fatal("mongodb connection uri is missing")
	} else {
		go database.StartDatabase(*db_uri)
	}

	<-database.UpOk

	alreadyOpenError := false
	var lastDateTime float64 = database.GetLastDateTime()

	for {
		file, err := os.Open(*squid_file)
		if err != nil {
			if !alreadyOpenError {
				logErr.Println(err)
				alreadyOpenError = true
			}
		} else {
			logInfo.Println(fmt.Sprintf("parsing file %s", file.Name()))
			alreadyOpenError = false
			err, lastDateTime = scraper.ParseFile(file, lastDateTime)
			file.Close()
			logErr.Println(err)
		}
		time.Sleep(3 * time.Second)
	}
}
