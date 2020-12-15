package squid

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
)

var logErr *log.Logger
var logInfo *log.Logger
var scriptPath string
var cutPath string

func SetLogOutput(w io.Writer) {
	logErr = log.New(w, "ERROR [squid]: ", log.LstdFlags|log.Lmsgprefix)
	logInfo = log.New(w, "INFO [squid]: ", log.LstdFlags|log.Lmsgprefix)
}

func SetReloadScript(script string) {
	scriptPath = script
}

func SetCutFile(cut string) {
	cutPath = cut
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

func Cut(user string) {
	// Append user to cut file
	if file, err := os.OpenFile(cutPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666); err != nil {
		logErr.Println(err)
	} else {
		file.WriteString(fmt.Sprintf("%s\n", user))
		Reload() // Reload Squid service
	}
}

