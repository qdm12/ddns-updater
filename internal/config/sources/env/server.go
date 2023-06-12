package env

import "github.com/qdm12/ddns-updater/internal/config/settings"

func (s *Source) readServer() (settings settings.Server, err error) {
	settings.RootURL = s.env.String("ROOT_URL")
	settings.Port, err = s.env.Uint16Ptr("LISTENING_PORT") // TODO change to address
	return settings, err
}
