package env

import (
	"github.com/qdm12/ddns-updater/internal/config/settings"
	"github.com/qdm12/gosettings/sources/env"
)

func (s *Source) readBackup() (settings settings.Backup, err error) {
	settings.Period, err = s.env.DurationPtr("BACKUP_PERIOD")
	if err != nil {
		return settings, err
	}

	settings.Directory = s.env.Get("BACKUP_DIRECTORY", env.ForceLowercase(false))
	return settings, nil
}
