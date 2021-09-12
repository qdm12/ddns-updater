package config

import (
	"net/url"
	"path"
	"strings"

	"github.com/containrrr/shoutrrr"
	"github.com/containrrr/shoutrrr/pkg/types"
	"github.com/qdm12/golibs/params"
)

type Shoutrrr struct {
	Addresses []string
	Params    types.Params
}

func (s *Shoutrrr) get(env params.Env) (warnings []string, err error) {
	s.Addresses, err = env.CSV("SHOUTRRR_ADDRESSES", params.CaseSensitiveValue())
	if err != nil {
		return warnings, err
	}

	// Retro-compatibility: GOTIFY_URL and GOTIFY_TOKEN
	gotifyURL, err := env.URL("GOTIFY_URL")
	if err != nil || gotifyURL != nil {
		const warning = "You should use the environment variable SHOUTRRR_ADDRESSES instead of GOTIFY_URL and GOTIFY_TOKEN"
		warnings = append(warnings, warning)
	}
	if err != nil {
		return warnings, err
	} else if gotifyURL != nil {
		gotifyToken, err := env.Get("GOTIFY_TOKEN", params.CaseSensitiveValue(),
			params.Compulsory(), params.Unset())
		if err != nil {
			return warnings, err
		}
		gotifyShoutrrrAddress := gotifyURLTokenToShoutrrr(gotifyURL, gotifyToken)
		s.Addresses = append(s.Addresses, gotifyShoutrrrAddress)
	}

	if _, err = shoutrrr.CreateSender(s.Addresses...); err != nil {
		return warnings, err // validation step
	}

	str, err := env.Get("SHOUTRRR_PARAMS", params.Default("title=DDNS Updater"), params.CaseSensitiveValue())
	if err != nil {
		return warnings, err
	}

	keyValues := strings.Split(str, ",")
	s.Params = make(map[string]string, len(keyValues))
	for _, keyValue := range keyValues {
		fields := strings.Split(keyValue, "=")
		key, value := fields[0], fields[1]
		s.Params[key] = value
	}

	return warnings, err
}

func gotifyURLTokenToShoutrrr(url *url.URL, token string) (address string) {
	hostAndPath := path.Join(url.Host, url.Path)
	address = "gotify://" + hostAndPath + "/" + token
	if url.Scheme == "http" {
		address += "?DisableTLS=Yes"
	}
	return address
}
