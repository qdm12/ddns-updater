package env

import (
	"github.com/qdm12/ddns-updater/internal/resolver"
)

func (s *Source) readResolver() (settings resolver.Settings, err error) {
	settings.Address = s.env.Get("RESOLVER_ADDRESS")
	settings.Timeout, err = s.env.Duration("RESOLVER_TIMEOUT")
	return settings, err
}
