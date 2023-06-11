package env

import (
	"github.com/qdm12/ddns-updater/internal/config/settings"
)

func (s *Source) readClient() (settings settings.Client, err error) {
	settings.Timeout, err = s.env.Duration("HTTP_TIMEOUT")
	return settings, err
}
