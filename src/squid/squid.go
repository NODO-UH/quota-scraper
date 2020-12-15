package squid

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
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

func Uncut(user string) error {
	if cutFile, err := os.OpenFile(cutPath, os.O_RDWR, 0666); err != nil { // Open cut file
		logErr.Println(err)
	} else if dataB, err := ioutil.ReadAll(cutFile); err != nil {
		logErr.Println(err)
		return err
	} else {
		// Remove line with user to uncut
		data := string(dataB)
		lines := strings.Split(data, "\n")
		var newLines []string
		for _, l := range lines {
			if user != l {
				newLines = append(newLines, l)
			}
		}
		// Rewrite other users to cut file
		if err := cutFile.Truncate(0); err != nil {
			logErr.Println(err)
			return err
		}
		if _, err := cutFile.Seek(0, 0); err != nil {
			logErr.Println(err)
			return err
		}
		if _, err := cutFile.WriteString(strings.Join(newLines, "\n")); err != nil {
			logErr.Println(err)
			return err
		}
		// Reload squid service
		Reload()
	}
	return nil
}
