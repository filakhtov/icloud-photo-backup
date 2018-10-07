package config

import (
	"fmt"
	"time"

	"github.com/BurntSushi/toml"
)

type Configuration interface {
	LogDir() string
	LockFile() string
	PollingInterval() time.Duration
	Source() string
	Destinations() []string
}

type configurationImpl struct {
	logDir, lockFile, source string
	destination              []string
	pollingInterval          time.Duration
}

func (c configurationImpl) LogDir() string {
	return c.logDir
}

func (c configurationImpl) LockFile() string {
	return c.lockFile
}

func (c configurationImpl) PollingInterval() time.Duration {
	return c.pollingInterval
}

func (c configurationImpl) Source() string {
	return c.source
}

func (c configurationImpl) Destinations() []string {
	return c.destination
}

func ParseConfiguration(configPath string) (config Configuration, err error) {
	var conf parsedConfiguration
	var metadata toml.MetaData

	if metadata, err = toml.DecodeFile(configPath, &conf); err != nil {
		err = fmt.Errorf("unable to parse configuration, error: %s", err)

		return
	}

	if len(metadata.Undecoded()) > 0 {
		err = fmt.Errorf("unable to parse configuration, unknown keys found: %q", metadata.Undecoded())

		return
	}

	for _, keyName := range []string{"logdir", "lockfile", "pollinginterval", "source", "destination"} {
		if !metadata.IsDefined(keyName) {
			err = fmt.Errorf("unable to parse configuration, missing \"%s\" configuration key", keyName)

			return
		}
	}

	config = configurationImpl{conf.LogDir, conf.LockFile, conf.Source, conf.Destination,
		conf.PollingInterval.Duration}

	return
}

type parsedConfiguration struct {
	LogDir, LockFile, Source string
	Destination              []string
	PollingInterval          duration
}

type duration struct {
	time.Duration
}

func (d *duration) UnmarshalText(text []byte) (err error) {
	if d.Duration, err = time.ParseDuration(string(text)); err != nil {
		err = fmt.Errorf("invalid duration format: %s", text)
	}

	return
}
