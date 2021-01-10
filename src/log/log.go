package log

import (
	"log"
	"os"
)

var (
	Info  *log.Logger
	Error *log.Logger
)

func init() {
	Info = log.New(os.Stdout, "INFO: ", log.Lmsgprefix)
	Error = log.New(os.Stderr, "ERROR: ", log.Lmsgprefix|log.Llongfile)
}
