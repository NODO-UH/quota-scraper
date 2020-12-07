package scraper

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/fsnotify/fsnotify"
)

var logErr *log.Logger
var logInfo *log.Logger

func init() {
	logErr = log.New(os.Stderr, "ERROR [scraper]: ", 1)
	logInfo = log.New(os.Stdout, "INFO [scraper]: ", 1)
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
		logErr.Println("unexpected squid line")
		return nil
	}

	// Check TCP_STATUS
	if words[3] == "TCP_DENIED/407" {
		return nil
	}

	sd := database.Quotalog{}
	if date_time, err := strconv.ParseFloat(words[0], 64); err != nil {
		logErr.Println(err)
		return nil
	} else {
		sd.DateTime = date_time
	}
	sd.User = words[7]
	if size, err := strconv.ParseInt(words[4], 10, 64); err != nil {
		logErr.Println(err)
		return nil
	} else {
		sd.Size = size
	}
	if ul, err := url.Parse(words[6]); err != nil {
		logErr.Println(err)
		return nil
	} else {
		sd.Url = ul
	}
	if from := net.ParseIP(words[2]); from == nil {
		logErr.Println(fmt.Sprintf("invalid ip %s", words[2]))
		return nil
	} else {
		sd.From = from.String()
	}
	return &sd
}

func parseLine(r *bufio.Reader) (*database.Quotalog, bool) {
	line_str, _, err := r.ReadLine()
	if err != nil {
		if err == io.EOF {
			return nil, true
		}
		logErr.Println(err)
		return nil, false
	}
	return parseQuotaLog(string(line_str)), false
}

func readLines(r *bufio.Reader) (sds []*database.Quotalog) {
	for sd, eof := parseLine(r); !eof; sd, eof = parseLine(r) {
		if sd != nil {
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
			database.AddQuotalog(v)
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
						database.AddQuotalog(v)
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
