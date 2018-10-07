package lockfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"sync/atomic"
)

type LockFile interface {
	Validate() error
	Destroy() error
}

func IsOrphanedError(err error) bool {
	return err != nil && err.Error() == orphanedError().Error()
}

func orphanedError() error {
	return fmt.Errorf("lock file exists, but no previous process is runnings")
}

func New(lockFilePath string) (lf LockFile, err error) {
	// e := doesLockFileExist(lockFilePath)
	// if e == nil {
	// 	err = fmt.Errorf("previous lock file exists")

	// 	return
	// } else if !os.IsNotExist(e) {
	// 	err = fmt.Errorf("unable to check previous lock file status, error: %s", e)

	// 	return
	// }

	if e := writePidToFile(lockFilePath); e != nil {
		err = e

		return
	}

	lf = lockFile{filePath: lockFilePath, isTampered: new(uint32)}

	return
}

type lockFile struct {
	filePath   string
	isTampered *uint32
}

func (lf lockFile) Validate() error {
	pid, err := readPidFromFile(lf.filePath)
	if err != nil {
		atomic.StoreUint32(lf.isTampered, 1)

		return fmt.Errorf("lock file validation failed, error: %s", err)
	}

	myPid := os.Getpid()
	if pid != myPid {
		atomic.StoreUint32(lf.isTampered, 1)

		return fmt.Errorf("lock file was modified externally: %d PID (in lock file) vs %d (my PID)", pid, myPid)
	}

	return nil
}

func (lf lockFile) Destroy() error {
	if atomic.LoadUint32(lf.isTampered) == 1 {
		return fmt.Errorf("not removing lock file %s because it was externally tampered with", lf.filePath)
	}

	if err := os.Remove(lf.filePath); err != nil {
		return fmt.Errorf("unable to remove %s lock file, error: %s", lf.filePath, err)
	}

	return nil
}

func writePidToFile(lockFilePath string) error {
	lockFileFd, err := os.Create(lockFilePath)
	if err != nil {
		return fmt.Errorf("unable to create %s lock file, error: %s", lockFilePath, err)
	}
	defer lockFileFd.Close()

	pid := strconv.Itoa(os.Getpid())
	if _, err := lockFileFd.Write([]byte(pid)); err != nil {
		return fmt.Errorf("unable to write pid to lock file, error: %s", lockFilePath)
	}

	return nil
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

func doesLockFileExist(lockFilePath string) error {
	_, err := os.Stat(lockFilePath)

	return err
}
