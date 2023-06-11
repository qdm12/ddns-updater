package env

import (
	"github.com/qdm12/ddns-updater/internal/config/settings"
)

func (s *Source) readPaths() (settings settings.Paths) {
	settings.DataDir = s.env.Get("DATADIR")
	return settings
}
