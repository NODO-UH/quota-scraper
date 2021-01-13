package scraper

import (
	"bufio"
	"errors"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/NODO-UH/quota-scraper/src/configuration"

	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/NODO-UH/quota-scraper/src/free"
	log "github.com/NODO-UH/quota-scraper/src/log"
	stats "github.com/NODO-UH/quota-scraper/src/prometheus"
	"github.com/fsnotify/fsnotify"
)

var userRegex *regexp.Regexp

func init() {
	userRegex, _ = regexp.Compile(".*@.*")
}

func parseQuotaLog(l string) *database.Quotalog {
	_words := strings.Split(l, " ")
	var words []string
	for _, w := range _words {
		if w != "" {
			words = append(words, w)
		}
	}
	if len(words) < 10 {
		return nil
	}

	// Check TCP_STATUS
	if discardStatus(words[3]) {
		return nil
	}

	sd := database.Quotalog{}
	date_time, err := strconv.ParseFloat(words[0], 64)
	if err != nil {
		log.Error.Println(err)
		return nil
	}
	sd.DateTime = date_time

	if userRegex.Match([]byte(words[7])) {
		sd.User = words[7]
	} else {
		return nil
	}
	size, err := strconv.ParseInt(words[4], 10, 64)
	if err != nil {
		log.Error.Println(err)
		return nil
	}
	sd.Size = size

	sd.Url = strings.TrimSpace(words[6])
	sd.From = strings.TrimSpace(words[2])
	return &sd
}

func parseLine(r *bufio.Reader) (*database.Quotalog, bool) {
	line_str, _, err := r.ReadLine()
	if err != nil {
		if err == io.EOF {
			return nil, true
		}
		log.Error.Println(err)
		return nil, false
	}
	return parseQuotaLog(string(line_str)), false
}

func readLines(r *bufio.Reader) (sds []*database.Quotalog) {
	for sd, eof := parseLine(r); !eof; sd, eof = parseLine(r) {
		stats.LogCountInc()
		if sd != nil {
			stats.LogValidInc()
			sds = append(sds, sd)
		}
	}
	return
}

func ParseFile(file *os.File, lastDateTime float64) (error, float64) {
	reader := bufio.NewReader(file)

	// Read initial lines
	for _, v := range readLines(reader) {
		if v.DateTime > lastDateTime {
			lastDateTime = v.DateTime
			if !free.IsFree(v.Url) {
				database.AddQuotalog(v)
			}
		}
	}

	// Suscribe to file changes
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err, lastDateTime
	}
	defer watcher.Close()

	if err = watcher.Add(file.Name()); err != nil {
		return err, lastDateTime
	}

	for {
		select {
		case event := <-watcher.Events:
			switch event.Op {
			case fsnotify.Write:
				for _, v := range readLines(reader) {
					if v.DateTime > lastDateTime {
						lastDateTime = v.DateTime
						if !free.IsFree(v.Url) {
							database.AddQuotalog(v)
						}
					}
				}
			default:
				return errors.New("unexpected watcher event"), lastDateTime
			}
		case err := <-watcher.Errors:
			return err, lastDateTime
		}
	}
}

func discardStatus(status string) bool {
	c := configuration.GetConfiguration()
	for _, i := range c.FreeTCPStatus {
		if status == i {
			return true
		}
	}
	return false
}
