package main

import (
	"bufio"
	"flag"
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

func main() {
	squid_file := flag.String("file", "squid.logs", "Path to squid file with logs")
	flag.Parse()

	logErr = log.New(os.Stderr, "ERROR: ", 1)
	logInfo = log.New(os.Stdout, "INFO: ", 1)

	logInfo.Println(fmt.Sprintf("squid file: %s", *squid_file))
	file, err := os.Open(*squid_file)
	if err != nil {
		logErr.Fatal(err)
	}

	reader := bufio.NewReader(file)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logErr.Fatal(err)
	}
	defer watcher.Close()

	done := make(chan bool)

	l := func() {
		for {
			select {
			case event := <-watcher.Events:
				switch event.Op {
				case fsnotify.Write:
					for _, v := range read_lines(reader) {
						logInfo.Println(v)
					}
				default:
					println("HERE")
				}
				// watch for errors
			case err := <-watcher.Errors:
				fmt.Println("ERROR", err)
			}
		}
	}

	// out of the box fsnotify can watch a single file, or a single directory
	if err = watcher.Add(*squid_file); err != nil {
		fmt.Println("ERROR", err)
	}
	go l()
	<-done

}
