package lockfile

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32dll  = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32dll.NewProc("LockFileEx")
	unlockFileEx = kernel32dll.NewProc("UnlockFileEx")
)

const (
	flagLockExclusive                     = 2
	flagLockFailImmediately               = 1
	errLockViolation        syscall.Errno = 33
)

func flock(h syscall.Handle, flags, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) error {
	r, _, err := lockFileEx.Call(uintptr(h), uintptr(flags), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)))
	if r == 0 {
		return err
	}

	return nil
}

func Flock(filePath *os.File) error {
	return flock(syscall.Handle(filePath.Fd()), flagLockExclusive|flagLockFailImmediately, 0, 1, 0, &syscall.Overlapped{})
}

func unlock(h syscall.Handle, reserved, locklow, lockhigh uint32, ol *syscall.Overlapped) error {
	r, _, err := unlockFileEx.Call(uintptr(h), uintptr(reserved), uintptr(locklow), uintptr(lockhigh), uintptr(unsafe.Pointer(ol)), 0)
	if r == 0 {
		return err
	}

	return nil
}

func Unlock(filePath *os.File) error {
	return unlock(syscall.Handle(filePath.Fd()), 0, 1, 0, &syscall.Overlapped{})
}
