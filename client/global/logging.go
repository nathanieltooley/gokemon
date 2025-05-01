package global

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nathanieltooley/gokemon/client/errorutils"
	"github.com/samber/lo"
)

type rollingFileWriter struct {
	FileDirectory string
	FileName      string
}

func NewRollingFileWriter(fileDir string, fileName string) rollingFileWriter {
	absFileDir, err := filepath.Abs(fileDir)
	if err != nil {
		panic(err)
	}

	// Create dir for log files if they dont exist
	// perms copied from docs, i dont know linux file perms numbers
	if err := os.MkdirAll(absFileDir, 0750); err != nil {
		panic(err)
	}

	return rollingFileWriter{
		FileDirectory: absFileDir,
		FileName:      fileName,
	}
}

// TODO: make this stuff runtime known through config file
const (
	mb         = 1000000
	kb         = 1000
	maxLogSize = 2.5 * mb
	maxLogs    = 2
	// maxLogSize = 0.5 * kb
)

func (w rollingFileWriter) getFullFilePath() string {
	return filepath.Join(w.FileDirectory, fmt.Sprintf("%s.log", w.FileName))
}

func (w rollingFileWriter) getLogs(pattern string) ([]string, error) {
	fileSystem, err := getFS(w.FileDirectory)
	if err != nil {
		return nil, err
	}

	// all log files ending in -*.log are archived logs
	// meaning they aren't getting updated
	logMatches, err := fs.Glob(fileSystem, pattern)
	if err != nil {
		return nil, err
	}

	return lo.Map(logMatches, func(log string, _ int) string {
		return filepath.Join(w.FileDirectory, log)
	}), nil
}

func (w rollingFileWriter) Write(b []byte) (n int, err error) {
	mainLogFile, err := os.OpenFile(w.getFullFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}

	stats, err := mainLogFile.Stat()
	if err != nil {
		// Not using defer because by the end of this function scope, mainLogFile most likely doesn't even point to the right thing anymore
		mainLogFile.Close()
		return 0, err
	}

	size := stats.Size()
	// if the current log file is small enough, just append to it
	if size < maxLogSize {
		defer mainLogFile.Close()
		return mainLogFile.Write(b)
	} else {
		// close since we are going to rename the main file
		mainLogFile.Close()
		w.updateLogIndices()
	}

	logMatches, err := w.getLogs(w.FileName + "-*.log")
	if err != nil {
		return 0, err
	}

	// At this point, there should be no main log file (they're all numbered: name-1.log, name-2.log, etc.)
	// So add in the main log file that will be there eventually
	numberOfLogFiles := len(logMatches) + 1

	// delete the last modified log file
	if numberOfLogFiles > maxLogs {
		difference := numberOfLogFiles - maxLogs
		var lastFile *os.File
		var latestFileIndex int64

		// delete an old file for as many times as necessary
		for range difference {
			for _, fileName := range logMatches {
				file_index := getLogIndex(w.FileName, fileName)

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

	// Append to a new log file
	mainLogFile, err = os.OpenFile(w.getFullFilePath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}
	defer mainLogFile.Close()

	return mainLogFile.Write(b)
}

func (w rollingFileWriter) indexedLog(fileName string, index int) string {
	return filepath.Join(w.FileDirectory, fmt.Sprintf("%s-%d.log", fileName, index))
}

func (w rollingFileWriter) updateLogIndices() error {
	logMatches, err := w.getLogs(w.FileName + "-*.log")
	if err != nil {
		return err
	}

	for _, log := range logMatches {
		index := getLogIndex(w.FileName, log)

		// get rid of messed up log files
		if index < 0 {
			if err := os.Remove(log); err != nil {
				return err
			}
		}

		index++
		// Add mod here since newly made files would conflict with old files
		// i.e if client-1.log gets renamed to client-2.log and client-2.log needs to be renamed to client-3.log
		// the new and old logs would conflict
		newFileName := w.indexedLog("mod-"+w.FileName, int(index))
		if err := os.Rename(log, newFileName); err != nil {
			return err
		}
	}

	modLogMatches, err := w.getLogs(fmt.Sprintf("mod-%s-*.log", w.FileName))
	if err != nil {
		return err
	}

	// Clean up mod prefixes
	for _, log := range modLogMatches {
		filenameOnly := filepath.Base(log)
		newFileName, _ := strings.CutPrefix(filenameOnly, "mod-")

		newFullPath := filepath.Join(filepath.Dir(log), newFileName)

		if err := os.Rename(log, newFullPath); err != nil {
			return err
		}
	}

	// Rename main log file
	if err := os.Rename(w.getFullFilePath(), w.indexedLog(w.FileName, 1)); err != nil {
		return err
	}

	return nil
}

func getLogIndex(baseFileName string, filePath string) int64 {
	fileName, _ := strings.CutSuffix(filepath.Base(filePath), ".log")
	indexStr, _ := strings.CutPrefix(fileName, baseFileName+"-")

	// TODO: Change this to actual error handling, even though the user shouldn't mess with this for no reason
	// and if they do they are dumb and stupid and i would dislike them
	index := errorutils.Must(strconv.ParseInt(indexStr, 10, 32))

	return index
}

func getFS(fileDir string) (fs.FS, error) {
	fileSystem := os.DirFS(fileDir)
	return fileSystem, nil
}
