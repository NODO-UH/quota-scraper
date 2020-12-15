package squid

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
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
	setUpUncutServer()
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

type UncutError struct {
	Message string
}

func handleUncut(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	e := json.NewEncoder(w)
	if req.Method != http.MethodPost {
		w.WriteHeader(http.StatusBadRequest)
		e.Encode(UncutError{
			Message: fmt.Sprintf("invalid method %s", req.Method),
		})
	}
	keys, ok := req.URL.Query()["username"]
	if !ok || len(keys[0]) < 1 {
		w.WriteHeader(http.StatusBadRequest)
		e.Encode(UncutError{
			Message: "missing username param",
		})
		return
	}
	username := keys[0]
	if err := Uncut(username); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		e.Encode(UncutError{
			Message: err.Error(),
		})
		return
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func setUpUncutServer() {
	http.HandleFunc("/uncut", handleUncut)
	err := http.ListenAndServeTLS(":2100", "server.crt", "server.key", nil)
	if err != nil {
		logErr.Println(err)
	}
}
