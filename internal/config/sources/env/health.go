package env

import (
	"github.com/qdm12/ddns-updater/internal/config/settings"
)

func (s *Source) ReadHealth() (settings settings.Health) {
	settings.ServerAddress = s.env.Get("HEALTH_SERVER_ADDRESS")
	settings.HealthchecksioUUID = s.env.Get("HEALTH_HEALTHCHECKSIO_UUID")
	return settings
}
