package scraper

import (
	"bufio"
	"errors"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/NODO-UH/quota-scraper/src/database"
	"github.com/fsnotify/fsnotify"
)

var logErr *log.Logger

func init() {
	logErr = log.New(os.Stderr, "ERROR [scraper]: ", 1)
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
	if dateTime, err := strconv.ParseFloat(words[0], 64); err != nil {
		logErr.Println(err)
		return nil
	} else {
		sd.DateTime = dateTime
	}
	sd.User = words[7]
	if size, err := strconv.ParseInt(words[4], 10, 64); err != nil {
		logErr.Println(err)
		return nil
	} else {
		sd.Size = size
	}
	sd.Url = strings.TrimSpace(words[6])
	sd.From = strings.TrimSpace(words[2])
	return &sd
}

func parseLine(r *bufio.Reader) (*database.Quotalog, bool) {
	lineStr, _, err := r.ReadLine()
	if err != nil {
		if err == io.EOF {
			return nil, true
		}
		logErr.Println(err)
		return nil, false
	}
	return parseQuotaLog(string(lineStr)), false
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

	// Subscribe to file changes
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
				isTruncated := true
				for _, v := range readLines(reader) {
					if v.DateTime > lastDateTime {
						lastDateTime = v.DateTime
						database.AddQuotalog(v)
						isTruncated = false
					}
				}
				if isTruncated {
					return nil, lastDateTime
				}
			default:
				return errors.New("unexpected watcher event"), lastDateTime
			}
		case err := <-watcher.Errors:
			return err, lastDateTime
		}
	}
}
