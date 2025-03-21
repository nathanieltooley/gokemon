package global

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/nathanieltooley/gokemon/client/errors"
)

type rollingFileWriter struct{}

const (
	mb         = 1000000
	kb         = 1000
	maxLogSize = 2.5 * kb
	maxLogs    = 2
	// maxLogSize = 0.5 * kb
)

func (w rollingFileWriter) Write(b []byte) (n int, err error) {
	mainLogFile, err := os.OpenFile("client.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}

	defer mainLogFile.Close()

	stats, err := mainLogFile.Stat()
	if err != nil {
		return 0, err
	}

	size := stats.Size()
	// if the current log file is small enough, just append to it
	if size < maxLogSize {
		return mainLogFile.Write(b)
	} else {
		updateLogIndices()
	}

	logMatches, err := getLogs("client-*.log")
	if err != nil {
		return 0, err
	}

	// if we got this far, we had to of opened client.log, meaning it exists
	numberOfLogFiles := len(logMatches) + 1

	// delete the last modified log file
	if numberOfLogFiles > maxLogs {
		difference := numberOfLogFiles - maxLogs
		var lastFile *os.File
		var latestFileIndex int64

		// delete an old file for as many times as necessary
		for range difference {
			for _, fileName := range logMatches {
				file_index := getLogIndex(fileName)

				if file_index > latestFileIndex {
					latestFileIndex = file_index
				} else {
					continue
				}

				file, err := os.Open(fileName)
				if err != nil {
					log.Print(err)
					continue
				}

				lastFile = file
			}

			if lastFile != nil {
				if err := os.Remove(lastFile.Name()); err != nil {
					return 0, err
				}
			}
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

func updateLogIndices() error {
	logMatches, err := getLogs("client-*.log")
	if err != nil {
		return err
	}

	for _, log := range logMatches {
		index := getLogIndex(log)

		// get rid of messed up log files
		if index <= 0 {
			if err := os.Remove(log); err != nil {
				return err
			}
		}

		index++
		// Add mod here since newly made files would conflict with old files
		// i.e if client-1.log gets renamed to client-2.log and client-2.log needs to be renamed to client-3.log
		// the new and old logs would conflict
		newFileName := fmt.Sprintf("mod-client-%d.log", index)
		if err := os.Rename(log, newFileName); err != nil {
			return err
		}
	}

	modLogMatches, err := getLogs("mod-client-*.log")
	if err != nil {
		return err
	}

	// Clean up mod prefixes
	for _, log := range modLogMatches {
		newName, _ := strings.CutPrefix(log, "mod-")

		if err := os.Rename(log, newName); err != nil {
			return err
		}
	}

	// Rename main log file
	if err := os.Rename("client.log", "client-1.log"); err != nil {
		return err
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

func getLogs(pattern string) ([]string, error) {
	fileSystem, err := getFS()
	if err != nil {
		return nil, err
	}

	// all log files ending in -*.log are archived logs
	// meaning they aren't getting updated
	logMatches, err := fs.Glob(fileSystem, pattern)
	if err != nil {
		return nil, err
	}

	return logMatches, nil
}

func getLogIndex(name string) int64 {
	fileName, _ := strings.CutSuffix(name, ".log")
	indexStr, _ := strings.CutPrefix(fileName, "client-")

	// TODO: Change this to actual error handling, even though the user shouldn't mess with this for no reason
	// and if they do they are dumb and stupid and i would dislike them
	index := errors.Must(strconv.ParseInt(indexStr, 10, 32))

	return index
}
