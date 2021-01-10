package free

import (
	"regexp"
	"strings"

	"github.com/NODO-UH/quota-scraper/src/database"
	log "github.com/NODO-UH/quota-scraper/src/log"
)

var regexFree *regexp.Regexp

func BuildRegexp() {
	log.Info.Println("loading free regexs")
	freeSet := database.GetAllFree()
	regexs := []string{}
	for _, fs := range freeSet {
		regexs = append(regexs, *fs.Regex)
	}
	freeStr := "(" + strings.Join(regexs, ")|(") + ")"
	log.Info.Printf("free regex: %s", freeStr)
	if regex, err := regexp.Compile(freeStr); err != nil {
		log.Error.Fatal(err)
	} else {
		regexFree = regex
	}
}

func IsFree(url string) bool {
	return regexFree.Match([]byte(url))
}
