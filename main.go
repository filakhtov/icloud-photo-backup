package main

import (
	"log"

	"github.com/filakhtov/icloud-photo-backup/config"
	"github.com/filakhtov/icloud-photo-backup/lockfile"
	"github.com/filakhtov/icloud-photo-backup/logger"
	"github.com/filakhtov/icloud-photo-backup/notification"
	"github.com/filakhtov/icloud-photo-backup/routine"
	"github.com/filakhtov/icloud-photo-backup/signal_handler"
)

func main() {
	configuration, err := config.ParseConfiguration("config.toml")
	if err != nil {
		log.Println(err)
		notification.Error("Configuration problem", err.Error())

		return
	}

	logWriter, err := logger.New(configuration.LogDir())
	if err != nil {
		log.Println(err)
		notification.Error("Logging problem", err.Error())

		return
	}
	defer logWriter.Destroy()
	log.Println("configured logger")

	signalHandler := signal_handler.New()
	defer signalHandler.Stop()
	log.Println("configured signal handler")

	lockFile, err := lockfile.New(configuration.LockFile())
	if err != nil {
		log.Println(err)
		notification.Error("Lock file error", err.Error())

		return
	}
	defer lockFile.Destroy()
	log.Println("lock file created")

	routine.New(configuration, signalHandler, lockFile).Run()
}
