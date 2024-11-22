package global

import (
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/nathanieltooley/gokemon/client/errors"
)

type rollingFileWriter struct{}

const (
	mb         = 1000000
	kb         = 1000
	maxLogSize = 2.5 * mb
	maxLogs    = 2
	// maxLogSize = 0.5 * kb
)

func (w rollingFileWriter) Write(b []byte) (n int, err error) {
	mainLogFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer mainLogFile.Close()
	if err != nil {
		return 0, err
	}

	stats, err := mainLogFile.Stat()
	if err != nil {
		return 0, err
	}

	size := stats.Size()
	// if the current log file is small enough, just append to it
	if size < maxLogSize {
		return mainLogFile.Write(b)
	} else {
		w.incrementLogs()
	}

	logMatches, err := getLogs()
	if err != nil {
		return 0, err
	}

	// if we got this far, we had to of opened client.log, meaning it exists
	numberOfLogFiles := len(logMatches) + 1

	// delete the last modified log file
	if numberOfLogFiles > maxLogs {
		var lastFile *os.File
		var latestModDate time.Time

		for _, fileName := range logMatches {
			file, err := os.Open(fileName)
			if err != nil {
				continue
			}

			stats, err := file.Stat()
			if err != nil {
				continue
			}

			modTime := stats.ModTime()
			if latestModDate.Compare(modTime) >= 1 {
				lastFile = file
				latestModDate = modTime
			}
		}

		if err := os.Remove(lastFile.Name()); err != nil {
			return 0, err
		}
	}

	mainLogFile.Close()

	// Append to a new log file
	mainLogFile, err = os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer mainLogFile.Close()

	return mainLogFile.Write(b)
}

func (w rollingFileWriter) incrementLogs() error {
	logMatches, err := getLogs()
	if err != nil {
		return err
	}

	if err := os.Rename("client.log", "client-1.log"); err != nil {
		return err
	}

	for _, log := range logMatches {
		fileName, _ := strings.CutSuffix(log, ".log")
		indexStr, _ := strings.CutPrefix(fileName, "client-")

		// TODO: Change this to actual error handling, even the user shouldn't mess with this for no reason
		// and if they do they are dumb and stupid and i would dislike them
		index := errors.Must(strconv.ParseInt(indexStr, 10, 32))

		// get rid of messed up log files
		if index <= 0 {
			// its not a big deal if they don't get deleted so just ignore the error
			_ = os.Remove(log)
		}

		index++
		newFileName := fmt.Sprintf("client-%d.log", index)
		if err := os.Rename(log, newFileName); err != nil {
			return err
		}
	}

	return nil
}

func getFS() (fs.FS, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	fileSystem := os.DirFS(workDir)
	return fileSystem, nil
}

func getLogs() ([]string, error) {
	fileSystem, err := getFS()
	if err != nil {
		return nil, err
	}

	// all log files ending in -*.log are archived logs
	// meaning they aren't getting updated
	logMatches, err := fs.Glob(fileSystem, "client-*.log")
	if err != nil {
		return nil, err
	}

	return logMatches, nil
}
