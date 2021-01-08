package squid

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/NODO-UH/quota-scraper/src/configuration"
	slog "github.com/NODO-UH/quota-scraper/src/log"
)

func logError(message string) {
	slog.Err(message, "[scraper]")
}

// Cut ...
func Cut(user string) error {
	config := configuration.GetConfiguration()

	if r, err := http.NewRequest("POST", fmt.Sprintf("%s/cut?group=%s&user=%s", *config.MasterCut, *config.Group, user), nil); err != nil {
		logError(err.Error())
		return err
	} else if resp, err := http.DefaultClient.Do(r); err != nil {
		logError(err.Error())
		return err
	} else if resp.StatusCode != http.StatusOK {
		err := errors.New("unexpected response status code")
		logError(err.Error())
		return err
	}
	return nil
}
