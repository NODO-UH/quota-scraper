package main

import (
	"flag"
	"os"
	"runtime"
	"time"

	"github.com/NODO-UH/quota-scraper/src/configuration"
	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/NODO-UH/quota-scraper/src/free"
	log "github.com/NODO-UH/quota-scraper/src/log"
	"github.com/NODO-UH/quota-scraper/src/prometheus"
	"github.com/NODO-UH/quota-scraper/src/scraper"
)

func main() {
	configFile := flag.String("config", "scraper-config.json", "path to JSON config file")
	flag.Parse()

	if configFile == nil {
		log.Error.Panicln("unknown configuration file")
	} else if err := configuration.LoadConfiguration(*configFile); err != nil {
		log.Error.Panicln("error loading configuration file")
	}

	prometheus.Start()

	config := configuration.GetConfiguration()

	log.Info.Printf("setting up with %d cores\n", *config.Cores)
	runtime.GOMAXPROCS(*config.Cores)

	log.Info.Printf("squid file: %s\n", *config.SquidFile)

	if *config.DbUri == "" {
		log.Error.Panicln("mongodb connection uri is missing")
	} else {
		go database.StartDatabase()
	}

	<-database.UpOk

	free.BuildRegexp()

	alreadyOpenError := false
	var lastDateTime float64 = database.GetLastDateTime()

	for {
		file, err := os.Open(*config.SquidFile)
		if err != nil {
			if !alreadyOpenError {
				log.Error.Println(err)
				alreadyOpenError = true
			}
		} else {
			log.Info.Printf("parsing file %s\n", file.Name())
			alreadyOpenError = false
			err, lastDateTime = scraper.ParseFile(file, lastDateTime)
			file.Close()
			log.Error.Println(err)
		}
		time.Sleep(3 * time.Second)
	}
}
