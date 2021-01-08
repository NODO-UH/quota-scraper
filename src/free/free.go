package free

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/NODO-UH/quota-scraper/src/database"
	slog "github.com/NODO-UH/quota-scraper/src/log"
)

var regexFree *regexp.Regexp

func logInfo(message string) {
	slog.Info(message, "[free]")
}

func logError(message string) {
	slog.Err(message, "[free]")
}

func logFatal(message string) {
	logError(message)
	os.Exit(1)
}

func BuildRegexp() {
	freeSet := database.GetAllFree()
	regexs := []string{}
	for _, fs := range freeSet {
		regexs = append(regexs, *fs.Regex)
	}
	freeStr := "(" + strings.Join(regexs, ")|(") + ")"
	logInfo(fmt.Sprintf("free regex: %s", freeStr))
	if regex, err := regexp.Compile(freeStr); err != nil {
		logFatal(err.Error())
	} else {
		regexFree = regex
	}
}

func IsFree(url string) bool {
	return regexFree.Match([]byte(url))
}
