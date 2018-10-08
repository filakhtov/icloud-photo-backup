package lockfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync/atomic"
)

type LockFile interface {
	Destroy() error
}

func New(lockFilePath string) (lf LockFile, err error) {
	fd, e := writePidToFile(lockFilePath)
	if e != nil {
		err = e

		return
	}

	lf = lockFile{filePath: lockFilePath, isTampered: new(uint32), fileFd: fd}

	return
}

type lockFile struct {
	filePath   string
	fileFd     *os.File
	isTampered *uint32
}

func (lf lockFile) Destroy() error {
	if atomic.LoadUint32(lf.isTampered) == 1 {
		return fmt.Errorf("not removing lock file %s because it was externally tampered with", lf.filePath)
	}

	if err := Unlock(lf.fileFd); err != nil {
		return fmt.Errorf("unable to unlock lock file %s, error: %s", lf.filePath, err)
	}

	if err := os.Remove(lf.filePath); err != nil {
		return fmt.Errorf("unable to remove %s lock file, error: %s", lf.filePath, err)
	}

	return nil
}

func writePidToFile(lockFilePath string) (*os.File, error) {
	lockFileFd, err := os.Create(lockFilePath)
	if err != nil {
		return nil, fmt.Errorf("unable to create %s lock file, error: %s", lockFilePath, err)
	}

	if err := Flock(lockFileFd); err != nil {
		lockFileFd.Close()

		return nil, fmt.Errorf("unable to lock %s lock file, error: %s", lockFilePath, err)
	}

	pid := strconv.Itoa(os.Getpid())
	if _, err := lockFileFd.Write([]byte(pid)); err != nil {
		lockFileFd.Close()

		return nil, fmt.Errorf("unable to write pid to lock file, error: %s", lockFilePath)
	}

	return lockFileFd, nil
}

func readPidFromFile(lockFile string) (int, error) {
	contents, err := ioutil.ReadFile(lockFile)
	if err != nil {
		return -1, fmt.Errorf("unable to read %s lock file: %s", lockFile, err)
	}

	pid, err := strconv.ParseInt(string(contents), 10, 64)
	if err != nil {
		return -1, fmt.Errorf("unable to convert pid file value %s to integer: %s", contents, err)
	}

	return int(pid), nil
}
