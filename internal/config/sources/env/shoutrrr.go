package env

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/qdm12/ddns-updater/internal/shoutrrr"
	"github.com/qdm12/gosettings/sources/env"
)

func (s *Source) readShoutrrr() (settings shoutrrr.Settings, err error) {
	settings.Addresses = s.env.CSV("SHOUTRRR_ADDRESSES", env.ForceLowercase(false))

	// Retro-compatibility: GOTIFY_URL and GOTIFY_TOKEN
	gotifyURLString := s.env.Get("GOTIFY_URL", env.ForceLowercase(false))
	if gotifyURLString != nil {
		s.handleDeprecated("GOTIFY_URL", "SHOUTRRR_ADDRESSES")
		gotifyURL, err := url.Parse(*gotifyURLString)
		if err != nil {
			return settings, fmt.Errorf("gotify URL: %w", err)
		}

		gotifyToken := s.env.String("GOTIFY_TOKEN", env.ForceLowercase(false))
		s.handleDeprecated("GOTIFY_TOKEN", "SHOUTRRR_ADDRESSES")
		gotifyShoutrrrAddress := gotifyURLTokenToShoutrrr(gotifyURL, gotifyToken)
		settings.Addresses = append(settings.Addresses, gotifyShoutrrrAddress)
	}

	paramsCSV := s.env.Get("SHOUTRRR_PARAMS", env.ForceLowercase(false))
	if paramsCSV != nil {
		keyValuePairs := strings.Split(*paramsCSV, ",")
		settings.Params = make(map[string]string, len(keyValuePairs))
		for _, keyValuePair := range keyValuePairs {
			fields := strings.Split(keyValuePair, "=")
			key, value := fields[0], fields[1]
			settings.Params[key] = value
		}
	}

	return settings, nil
}

func gotifyURLTokenToShoutrrr(url *url.URL, token string) (address string) {
	hostAndPath := path.Join(url.Host, url.Path)
	address = "gotify://" + hostAndPath + "/" + token
	if url.Scheme == "http" {
		address += "?DisableTLS=Yes"
	}
	return address
}
