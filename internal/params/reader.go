package params

import (
	"io/fs"
	"os"

	"github.com/qdm12/golibs/params"
)

type Reader struct {
	logger    Logger
	env       envInterface
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
		env:       params.New(),
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
	}
}
