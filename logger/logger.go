package logger

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Logger struct {
	destination *os.File
}

func (logger Logger) Destroy() error {
	log.SetOutput(os.Stdout)

	if err := logger.destination.Sync(); err != nil {
		return fmt.Errorf("unable to synchronize log file to disk")
	}

	if err := logger.destination.Close(); err != nil {
		return fmt.Errorf("unable to gracefuly close the log file")
	}

	logger.destination = nil

	return nil
}

func New(logDir string) (logger Logger, err error) {
	if logDir[len(logDir)-1:] != string(os.PathSeparator) {
		logDir += string(os.PathSeparator)
	}

	time := time.Now()
	logFile := logDir + fmt.Sprintf("%04d-%02d-%02d_%02d-%02d-%02d_%d.log", time.Year(), time.Month(), time.Day(),
		time.Hour(), time.Minute(), time.Second(), time.Nanosecond())

	file, err := os.Create(logFile)
	if err != nil {
		return
	}

	logger.destination = file
	//writer := io.MultiWriter(os.Stdout, file)
	//log.SetOutput(writer)
	log.SetOutput(file)

	return
}
