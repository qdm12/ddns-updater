package params

import (
	"io/fs"
	"os"
)

type Reader struct {
	logger Logger
	// environmentConfig is the configuration given with the CONFIG
	// environment variable, which is empty if it is not set.
	environmentConfig        string
	persistEnvironmentConfig bool
	readFile                 func(filename string) ([]byte, error)
	writeFile                func(filename string, data []byte, perm fs.FileMode) (err error)
}

type Logger interface {
	Info(s string)
	Debug(s string)
}

func NewReader(logger Logger, environmentConfig string, persistEnvironmentConfig bool) *Reader {
	return &Reader{
		logger:                   logger,
		environmentConfig:        environmentConfig,
		persistEnvironmentConfig: persistEnvironmentConfig,
		readFile:                 os.ReadFile,
		writeFile:                os.WriteFile,
	}
}
