package log

import (
	"fmt"
	"io"
	"log"
	"os"
)

var logErr *log.Logger
var logInfo *log.Logger

func init() {
	logErr = log.New(os.Stdout, "ERROR ", log.LstdFlags|log.Lmsgprefix)
	logInfo = log.New(os.Stdout, "INFO ", log.LstdFlags|log.Lmsgprefix)
}

// SetOutErr ...
func SetOutErr(w io.Writer) {
	logErr.SetOutput(w)
}

// SetOutInfo ...
func SetOutInfo(w io.Writer) {
	logInfo.SetOutput(w)
}

// Err ...
func Err(message, module string) {
	logErr.Println(fmt.Sprintf("%s %s", module, message))
}

// Info ...
func Info(message, module string) {
	logInfo.Println(fmt.Sprintf("%s %s", module, message))
}
