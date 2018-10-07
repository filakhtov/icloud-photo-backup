package exif

import (
	"bytes"
	"fmt"
	"os/exec"
	"syscall"
	"time"
)

func GetExifDate(filename string) (time.Time, error) {
	var out, errOut bytes.Buffer

	cmd := exec.Command("exiftool.exe", "-T", "-CreateDate", "-d", "%Y-%m-%dT%H:%M:%S", filename)
	cmd.Stdout = &out
	cmd.Stderr = &errOut
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	if err := cmd.Run(); err != nil {
		return time.Now(), fmt.Errorf("%s (%s)", errOut.String(), err)
	}

	date, err := parseExifDate(out.String())
	if err != nil {
		return time.Now(), err
	}

	return date, nil
}

func parseExifDate(dateString string) (time.Time, error) {
	if len(dateString) < 19 {
		return time.Now(), fmt.Errorf("invalid date/time format \"%s\"", dateString)
	}

	date, err := time.Parse(time.RFC3339[:19], dateString[:19])
	if err != nil {
		return time.Now(), err
	}

	return date, nil
}
