package config

import (
	"time"

	"github.com/qdm12/golibs/params"
)

type Backup struct {
	Period    time.Duration
	Directory string
}

func (b *Backup) get(env params.Env) (err error) {
	b.Period, err = env.Duration("BACKUP_PERIOD", params.Default("0"))
	if err != nil {
		return err
	}
	b.Directory, err = env.Path("BACKUP_DIRECTORY", params.Default("./data"))
	return err
}
