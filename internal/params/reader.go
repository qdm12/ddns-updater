package params

import (
	"io/fs"
	"io/ioutil"
	"os"

	"github.com/qdm12/ddns-updater/internal/settings"
	"github.com/qdm12/golibs/params"
)

type Reader interface {
	JSONSettings(filePath string) (allSettings []settings.Settings, warnings []string, err error)
}

type reader struct {
	env       envInterface
	readFile  func(filename string) ([]byte, error)
	writeFile func(filename string, data []byte, perm fs.FileMode) (err error)
}

func NewReader() Reader {
	return &reader{
		env:       params.New(),
		readFile:  ioutil.ReadFile,
		writeFile: os.WriteFile,
	}
}
