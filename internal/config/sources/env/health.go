package env

import (
	"github.com/qdm12/ddns-updater/internal/config/settings"
)

func (s *Source) ReadHealth() (settings settings.Health) {
	settings.ServerAddress = s.env.Get("HTTP_TIMEOUT")
	return settings
}
