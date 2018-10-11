package routine

import (
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/filakhtov/icloud-photo-backup/config"
	"github.com/filakhtov/icloud-photo-backup/exif"
	"github.com/filakhtov/icloud-photo-backup/lockfile"
	"github.com/filakhtov/icloud-photo-backup/notification"
	"github.com/filakhtov/icloud-photo-backup/signal_handler"
)

type Routine interface {
	Run()
}

type routine struct {
	configuration config.Configuration
	signalHandler signal_handler.SignalHandler
	lockFile      lockfile.LockFile
}

var ignoredFileError = errors.New("not touching blacklisted file")

func New(cfg config.Configuration, sh signal_handler.SignalHandler, lf lockfile.LockFile) Routine {
	return routine{configuration: cfg, signalHandler: sh, lockFile: lf}
}

func (r routine) Run() {
	var successes, failures int64

	for {
		if !r.shouldRun() {
			return
		}

		s, f := r.processDirectory()
		successes += s
		failures += f

		if s > 0 {
			continue
		}

		log.Printf("Processing finished. Successes: %d, Failures: %d\n", successes, failures)

		if failures > 0 {
			notification.Warning("Backup completed with error",
				fmt.Sprintf("Successfully copied: %d\nFailed to copy: %d", successes, failures))
		} else if successes > 0 {
			notification.Info("Backup completed successfully",
				fmt.Sprintf("Successfully copied: %d\nFailed to copy: %d", successes, failures))
		}

		successes = 0
		failures = 0

		r.sleep()
	}
}

func (r routine) processDirectory() (int64, int64) {
	files, err := ioutil.ReadDir(r.configuration.Source())
	if err != nil {
		log.Printf("unable to access source directory, error: %s\n", err)
		notification.Warning("Backup failed", err.Error())

		return 0, 0
	}

	log.Printf("Opened source directory: %s\n", r.configuration.Source())

	return r.processFiles(r.removeDuplicates(files))
}

func (r routine) removeDuplicates(files []os.FileInfo) []os.FileInfo {
	var unique []os.FileInfo

	for _, file := range files {
		if !r.shouldRun() {
			return []os.FileInfo{}
		}

		filePath := filepath.Join(r.configuration.Source(), file.Name())
		filePathAbs, err := filepath.Abs(filePath)
		if err != nil {
			log.Printf("unable to get absolute file path for %s, error: %s\n", file.Name(), err)

			return []os.FileInfo{}
		}

		hasDuplicate, err := r.hasDuplicateHeic(filePathAbs)
		if err != nil {
			log.Printf("unable to check if %s JPEG has a HEIC pair, error: %s\n", file.Name(), err)

			return []os.FileInfo{}
		}

		if hasDuplicate {
			if err := os.Remove(filePathAbs); err != nil {
				log.Printf("unable to remove JPEG duplicate of HEIC %s, error: %s\n", file.Name(), err)

				return []os.FileInfo{}
			}

			log.Printf("found duplicate JPEG/HEIC pair and removed JPEG: %s\n", file.Name())
		} else {
			unique = append(unique, file)
		}
	}

	return unique
}

func (r routine) processFiles(files []os.FileInfo) (int64, int64) {
	var successes, failures int64

	for _, file := range files {
		if !r.shouldRun() {
			return successes, failures
		}

		err := r.processFile(file.Name())
		if err == ignoredFileError {
			continue
		}

		if err == nil {
			successes++
		} else {
			failures++
		}
	}

	return successes, failures
}

func (r routine) processFile(fileName string) error {
	if isBlacklisted(fileName) {
		log.Printf("not touching blacklisted file %s", fileName)

		return ignoredFileError
	}

	log.Printf("processing file: %s\n", fileName)

	filePath := filepath.Join(r.configuration.Source(), fileName)
	filePathAbs, err := filepath.Abs(filePath)
	if err != nil {
		log.Printf("unable to get absolute file path for %s, error: %s\n", fileName, err)

		return err
	}

	destName, err := r.getDestinationFileName(filePathAbs)
	if err != nil {
		log.Println(err)

		return err
	}

	log.Printf("Destination name for %s is %s\n", fileName, destName)

	if err := backupFile(filePathAbs, destName, r.configuration.Destination()); err != nil {
		log.Println(err)

		return err
	}

	log.Printf("processing file is finished: %s\n", fileName)

	return nil
}

func isBlacklisted(fileName string) bool {
	switch strings.ToLower(fileName) {
	case
		"thumbs.db",
		"desktop.ini":
		return true
	}
	return false
}

func backupFile(srcAbsPath string, destName string, destDir string) error {
	destPath := filepath.Join(destDir, destName)
	destAbsPath, err := filepath.Abs(destPath)
	if err != nil {
		return fmt.Errorf("unable to get destination file absolute path for %s, error: %s", destName, err)
	}

	if err := copyFile(srcAbsPath, destAbsPath); err != nil {
		return err
	}

	log.Printf("successfully backed up %s to %s", srcAbsPath, destAbsPath)

	if err := removeDuplicate(destAbsPath); err != nil {
		return fmt.Errorf("unable to check for %s duplicates, error: %s", destAbsPath, err)
	}

	if err := os.Remove(srcAbsPath); err != nil {
		return fmt.Errorf("unable to remove source file %s after backup, error: %s", srcAbsPath, err)
	}

	return nil
}

func removeDuplicate(filePath string) error {
	if isHeicFile(filePath) {
		duplicateName := swapExtension(filePath, ".jpg")
		_, err := os.Stat(duplicateName)
		if os.IsNotExist(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if err := os.Remove(duplicateName); err != nil {
			return err
		}

		log.Printf("removed duplicate JPEG %s file for %s HEIC file", duplicateName, filePath)
	}

	if isJpegFile(filePath) {
		duplicateName := swapExtension(filePath, ".heic")
		_, err := os.Stat(duplicateName)
		if os.IsNotExist(err) {
			return nil
		}

		if err != nil {
			return err
		}

		if err := os.Remove(filePath); err != nil {
			return err
		}

		log.Printf("removed duplicate JPEG %s file for %s HEIC file", filePath, duplicateName)
	}

	return nil
}

func (r routine) hasDuplicateHeic(srcAbsPath string) (bool, error) {
	if !isJpegFile(srcAbsPath) {
		return false, nil
	}

	srcAbsPathHeic := swapExtension(srcAbsPath, ".heic")
	_, err := os.Stat(srcAbsPathHeic)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}

	dateJpg, err := exif.GetExifDate(srcAbsPath)
	if err != nil {
		return false, fmt.Errorf("%s - %s", srcAbsPath, err)
	}

	dateHeic, err := exif.GetExifDate(srcAbsPathHeic)
	if err != nil {
		return false, fmt.Errorf("%s - %s", srcAbsPathHeic, err)
	}

	if dateHeic != dateJpg {
		return false, nil
	}

	return true, nil
}

func isJpegFile(path string) bool {
	return hasExtension(path, ".jpg")
}

func isHeicFile(path string) bool {
	return hasExtension(path, ".heic")
}

func hasExtension(fileName string, extension string) bool {
	return strings.ToLower(filepath.Ext(fileName)) == strings.ToLower(extension)
}

func swapExtension(path string, newExtension string) string {
	return removeExtension(path) + newExtension
}

func removeExtension(path string) string {
	oldExtensionLength := len(filepath.Ext(path))
	pathLengthWitoutExtension := len(path) - oldExtensionLength

	return path[:pathLengthWitoutExtension]
}

func copyFile(srcAbsPath string, destAbsPath string) error {
	srcFile, err := os.Open(srcAbsPath)
	if err != nil {
		return fmt.Errorf("unable to open source file %s, error: %s", srcAbsPath, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(destAbsPath)
	if err != nil {
		return fmt.Errorf("unable to open destination file %s, error: %s", destAbsPath, err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, srcFile); err != nil {
		return fmt.Errorf("unable to copy %s to %s, error: %s", srcAbsPath, destAbsPath, err)
	}

	if err := destFile.Sync(); err != nil {
		return fmt.Errorf("unable to synchronize destination file %s to disk, error: %s", destAbsPath, err)
	}

	return nil
}

func (r routine) getDestinationFileName(filePathAbs string) (string, error) {
	date, err := exif.GetExifDate(filePathAbs)
	if err != nil {
		log.Printf("unable to obtain EXIF date from %s file, error: %s", filePathAbs, err)

		finfo, err := os.Stat(filePathAbs)
		if err != nil {
			return "", fmt.Errorf("unable to obtain EXIF or modification date for %s, error: %s", filePathAbs, err)
		}

		date = finfo.ModTime()
	}

	sum, err := fileCrc32(filePathAbs)
	if err != nil {
		return "", fmt.Errorf("unable to compute file has for %s, error: %s", filePathAbs, err)
	}

	return fmt.Sprintf("%04d%02d%02d_%02d%02d%02d-%x%s", date.Year(), date.Month(), date.Day(), date.Hour(),
		date.Minute(), date.Second(), sum, filepath.Ext(filePathAbs)), nil
}

func fileCrc32(fileName string) (uint32, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	hash := crc32.NewIEEE()
	if _, err := io.Copy(hash, f); err != nil {
		log.Fatal(err)
	}

	return hash.Sum32(), nil
}

func (r routine) shouldRun() bool {
	if !r.signalHandler.ShouldContinue() {
		log.Println("OS interrupt received, terminating")

		return false
	}

	return true
}

func (r routine) sleep() {
	go func(cfg config.Configuration, sh signal_handler.SignalHandler) {
		time.Sleep(cfg.PollingInterval())
		sh.Continue()
	}(r.configuration, r.signalHandler)

	r.signalHandler.Wait()
}
