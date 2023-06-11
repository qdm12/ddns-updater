package params

import (
	"io/fs"
	"os"
)

type Reader struct {
	logger    Logger
	readFile  func(filename string) ([]byte, error)
	writeFile func(filename string, data []byte, perm fs.FileMode) (err error)
}

type Logger interface {
	Info(s string)
	Debug(s string)
}

func NewReader(logger Logger) *Reader {
	return &Reader{
		logger:    logger,
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
	}
}
