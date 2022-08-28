package params

import (
	"io/fs"
	"os"

	"github.com/qdm12/golibs/logging"
	"github.com/qdm12/golibs/params"
)

type Reader struct {
	logger    logging.Logger
	env       envInterface
	readFile  func(filename string) ([]byte, error)
	writeFile func(filename string, data []byte, perm fs.FileMode) (err error)
}

func NewReader(logger logging.Logger) *Reader {
	return &Reader{
		logger:    logger,
		env:       params.New(),
		readFile:  os.ReadFile,
		writeFile: os.WriteFile,
	}
}
