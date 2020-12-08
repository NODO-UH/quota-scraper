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
	flag.Parse()

	logInfo.Printf("setting up with %d cores", cores)
	runtime.GOMAXPROCS(*cores)

	logInfo.Println(fmt.Sprintf("squid file: %s", *squid_file))

	if *db_uri == "" {
		logErr.Fatal("mongodb connection uri is missing")
	} else {
		go database.StartDatabase(*db_uri)
	}

	<-database.UpOk

	alreadyOpenError := false
	var lastDateTime float64 = 0

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
			logErr.Println(err)
		}
		time.Sleep(3 * time.Second)
	}
}
