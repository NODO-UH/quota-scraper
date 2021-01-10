package squid

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/NODO-UH/quota-scraper/src/configuration"
	log "github.com/NODO-UH/quota-scraper/src/log"
)

// Cut ...
func Cut(user string) error {
	config := configuration.GetConfiguration()

	if r, err := http.NewRequest("POST", fmt.Sprintf("%s/cut?group=%s&user=%s", *config.MasterCut, *config.Group, user), nil); err != nil {
		log.Error.Println(err)
		return err
	} else if resp, err := http.DefaultClient.Do(r); err != nil {
		log.Error.Println(err)
		return err
	} else if resp.StatusCode != http.StatusOK {
		err := errors.New("unexpected response status code")
		log.Error.Println(err)
		return err
	}
	return nil
}
