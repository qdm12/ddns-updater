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
	env       params.Env
	readFile  func(filename string) ([]byte, error)
	writeFile func(filename string, data []byte, perm fs.FileMode) (err error)
}

func NewReader() Reader {
	return &reader{
		env:       params.NewEnv(),
		readFile:  ioutil.ReadFile,
		writeFile: os.WriteFile,
	}
}
