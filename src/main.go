package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/NODO-UH/quota-scraper/src/scraper"
	"github.com/NODO-UH/quota-scraper/src/squid"
)

var logErr *log.Logger
var logInfo *log.Logger
var hostName string

func init() {
	hostName, _ = os.Hostname()
}

type ScraperConfig struct {
	SquidFile   *string
	DbUri       *string
	Cores       *int
	LogsPath    *string
	ScraperId   *string
	CutFile     *string
	ReloadSquid *string
}

func ConfigFromFile(path string, flagConfig ScraperConfig) *ScraperConfig {
	config := &ScraperConfig{}
	if configFile, err := os.Open(path); err != nil {
		logErr.Println(err)
	} else {
		jsonDecoder := json.NewDecoder(configFile)
		if err = jsonDecoder.Decode(config); err != nil {
			logErr.Println(err)
		}
	}

	if config.SquidFile == nil {
		config.SquidFile = flagConfig.SquidFile
	}
	if config.DbUri == nil {
		config.DbUri = flagConfig.DbUri
	}
	if config.Cores == nil {
		config.Cores = flagConfig.Cores
	}
	if config.LogsPath == nil {
		config.LogsPath = flagConfig.LogsPath
	}
	if config.ScraperId == nil {
		config.ScraperId = flagConfig.ScraperId
	}
	if config.CutFile == nil {
		config.CutFile = flagConfig.CutFile
	}
	if config.ReloadSquid == nil {
		config.ReloadSquid = flagConfig.ReloadSquid
	}

	return config
}

func main() {
	configFile := flag.String("config", "scraper-config.json", "path to JSON config file")
	squidFile := flag.String("file", "squid.logs", "Path to squid file with logs")
	dbUri := flag.String("db-uri", "", "MongoDB Connection URI")
	cores := flag.Int("cores", runtime.NumCPU(), "max number of cores")
	logsPath := flag.String("logs", "squid-parser.logs", "path to file for logs")
	scraperId := flag.String("id", hostName, "unique id between all quota-scraper instances")
	cutFile := flag.String("cut-file", "cut.list", "file to insert over quota users")
	reloadSquid := flag.String("reload", "reload.sh", "script for reload Squid service")
	flag.Parse()

	config := ConfigFromFile(*configFile, ScraperConfig{
		SquidFile:   squidFile,
		DbUri:       dbUri,
		Cores:       cores,
		LogsPath:    logsPath,
		ScraperId:   scraperId,
		CutFile:     cutFile,
		ReloadSquid: reloadSquid,
	})

	if logsFile, err := os.OpenFile(*config.LogsPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
		logErr.Fatal(err)
	} else {
		logErr = log.New(logsFile, "ERROR [main] ", log.LstdFlags|log.Lmsgprefix)
		logInfo = log.New(logsFile, "INFO [main] ", log.LstdFlags|log.Lmsgprefix)
		database.SetLogOutput(logsFile)
		scraper.SetLogOutput(logsFile)
		squid.SetLogOutput(logsFile)
	}

	logInfo.Printf("setting up with %d cores", *config.Cores)
	runtime.GOMAXPROCS(*config.Cores)

	logInfo.Println(fmt.Sprintf("squid file: %s", *config.SquidFile))

	if *config.DbUri == "" {
		logErr.Fatal("mongodb connection uri is missing")
	} else {
		go database.StartDatabase(*config.DbUri, *scraperId)
	}

	<-database.UpOk

	// Set path of script for reload Squid service
	squid.SetReloadScript(*config.ReloadSquid)
	squid.SetCutFile(*config.CutFile)

	alreadyOpenError := false
	var lastDateTime float64 = database.GetLastDateTime(*config.ScraperId)

	for {
		file, err := os.Open(*config.SquidFile)
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
