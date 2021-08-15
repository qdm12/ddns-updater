package config

import (
	"fmt"
	"path/filepath"

	"github.com/qdm12/golibs/params"
)

type Paths struct {
	DataDir string
	JSON    string // obtained from DataDir
}

func (p *Paths) get(env params.Env) (err error) {
	p.DataDir, err = env.Path("DATADIR", params.Default("./data"))
	if err != nil {
		return fmt.Errorf("%w: for environment variable DATADIR", err)
	}

	p.JSON = filepath.Join(p.DataDir, "config.json")
	return nil
}
