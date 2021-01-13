package configuration

import (
	"encoding/json"
	"os"

	log "github.com/NODO-UH/quota-scraper/src/log"
)

type ScraperConfig struct {
	Id            *string
	SquidFile     *string
	DbUri         *string
	Cores         *int
	MasterCut     *string
	Group         *string
	FreeTCPStatus []string `json:"freeTcpStatus"`
}

var configuration *ScraperConfig

// LoadConfiguration ...
func LoadConfiguration(path string) error {
	configuration = &ScraperConfig{}
	configFile, err := os.Open(path)
	if err != nil {
		log.Error.Println(err)
		return err
	}
	jsonDecoder := json.NewDecoder(configFile)
	if err = jsonDecoder.Decode(configuration); err != nil {
		log.Error.Println(err)
		return err
	}

	return nil
}

// GetConfiguration ...
func GetConfiguration() ScraperConfig {
	return *configuration
}
