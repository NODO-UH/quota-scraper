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

	"github.com/fsnotify/fsnotify"
)

var logErr *log.Logger
var logInfo *log.Logger

func init() {
	logErr = log.New(os.Stderr, "ERROR: ", 1)
	logInfo = log.New(os.Stdout, "INFO: ", 1)
}

type SquidData struct {
	DateTime float64
	User     string
	Size     int64
	Url      *url.URL
	From     net.IP
}

func (sd SquidData) String() string {
	return fmt.Sprintf("DT: %f, User: %s, Size: %d, Url: %s, From: %s", sd.DateTime, sd.User, sd.Size, sd.Url, sd.From)
}

func parse_SquidData(l string) *SquidData {
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
	sd := SquidData{}
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
		sd.From = from
	}
	return &sd
}

func parse_line(r *bufio.Reader) (*SquidData, bool) {
	line_str, _, err := r.ReadLine()
	if err != nil {
		if err == io.EOF {
			return nil, true
		}
		logErr.Println(err)
		return nil, false
	}
	return parse_SquidData(string(line_str)), false
}

func read_lines(r *bufio.Reader) (sds []*SquidData) {
	for sd, eof := parse_line(r); !eof; sd, eof = parse_line(r) {
		sds = append(sds, sd)
	}
	return
}

func ParseFile(file *os.File, lastDateTime float64) (error, float64) {
	reader := bufio.NewReader(file)

	// Read initial lines
	for _, v := range read_lines(reader) {
		if v.DateTime > lastDateTime {
			logInfo.Println(v)
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
				for _, v := range read_lines(reader) {
					if v.DateTime > lastDateTime {
						lastDateTime = v.DateTime
						// Parsed line
						logInfo.Println(v)
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
