package squid

import (
	"io"
	"log"
	"os/exec"
)

var logErr *log.Logger
var logInfo *log.Logger
var scriptPath string

func SetLogOutput(w io.Writer) {
	logErr = log.New(w, "ERROR [squid]: ", log.LstdFlags|log.Lmsgprefix)
	logInfo = log.New(w, "INFO [squid]: ", log.LstdFlags|log.Lmsgprefix)
}

func SetReloadScript(script string) {
	scriptPath = script
}

func Reload() {
	logInfo.Println("reload squid service")
	cmd := exec.Command("/bin/bash", scriptPath)
	cmd.Stderr = logErr.Writer()
	_, err := cmd.Output()
	if err != nil {
		logErr.Println(err)
	}
}
