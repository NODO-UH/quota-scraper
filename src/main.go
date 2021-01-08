package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/NODO-UH/quota-scraper/src/configuration"
	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/NODO-UH/quota-scraper/src/free"
	slog "github.com/NODO-UH/quota-scraper/src/log"
	"github.com/NODO-UH/quota-scraper/src/prometheus"
	"github.com/NODO-UH/quota-scraper/src/scraper"
)

var hostName string
var module string

func logInfo(message string) {
	slog.Info(message, "[main]")
}

func logError(message string) {
	slog.Err(message, "[main]")
}

func main() {
	configFile := flag.String("config", "scraper-config.json", "path to JSON config file")
	logsPath := flag.String("logs", "squid-parser.logs", "path to file for logs")
	flag.Parse()

	if f, err := os.OpenFile(*logsPath, os.O_CREATE|os.O_RDWR, 0666); err != nil {
		logError(err.Error())
	} else {
		slog.SetOutErr(f)
		slog.SetOutInfo(f)
	}

	if configFile == nil {
		panic("unknown configuration file")
	}

	// Load configuration
	if err := configuration.LoadConfiguration(*configFile); err != nil {
		panic("error loading configuration file")
	}

	prometheus.Start()

	config := configuration.GetConfiguration()

	logInfo(fmt.Sprintf("setting up with %d cores", *config.Cores))
	runtime.GOMAXPROCS(*config.Cores)

	logInfo(fmt.Sprintf("squid file: %s", *config.SquidFile))

	if *config.DbUri == "" {
		logError("mongodb connection uri is missing")
	} else {
		go database.StartDatabase()
	}

	<-database.UpOk

	logInfo("loading free regexs")
	free.BuildRegexp()

	alreadyOpenError := false
	var lastDateTime float64 = database.GetLastDateTime()

	for {
		file, err := os.Open(*config.SquidFile)
		if err != nil {
			if !alreadyOpenError {
				logError(err.Error())
				alreadyOpenError = true
			}
		} else {
			logInfo(fmt.Sprintf("parsing file %s", file.Name()))
			alreadyOpenError = false
			err, lastDateTime = scraper.ParseFile(file, lastDateTime)
			file.Close()
			logError(err.Error())
		}
		time.Sleep(3 * time.Second)
	}
}
