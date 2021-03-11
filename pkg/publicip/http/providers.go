package http

import (
	"errors"
	"fmt"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider string

const (
	Google   Provider = "google"
	Ifconfig Provider = "ifconfig"
	Ipify    Provider = "ipify"
	Ipinfo   Provider = "ipinfo"
	Noip     Provider = "noip"
	Opendns  Provider = "opendns"
)

func ListProviders() []Provider {
	return []Provider{
		Google,
		Ifconfig,
		Ipify,
		Ipinfo,
		Noip,
		Opendns,
	}
}

var (
	ErrUnknownProvider   = errors.New("unknown provider")
	ErrProviderIPVersion = errors.New("provider does not support IP version")
)

func ValidateProvider(provider Provider, version ipversion.IPVersion) error {
	for _, possible := range ListProviders() {
		if provider == possible {
			_, ok := provider.url(version)
			if !ok {
				return fmt.Errorf("%w: %q for version %s",
					ErrProviderIPVersion, provider, version.String())
			}
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
}

func (provider Provider) url(version ipversion.IPVersion) (url string, ok bool) {
	switch version {

	case ipversion.IP4:
		switch provider {
		case Ipify:
			url = "https://api.ipify.org"
		case Noip:
			url = "http://ip1.dynupdate.no-ip.com"
		}

	case ipversion.IP6:
		switch provider {
		case Ipify:
			url = "https://api6.ipify.org"
		case Noip:
			url = "http://ip1.dynupdate6.no-ip.com"
		}

	case ipversion.IP4or6:
		switch provider {
		case Google:
			url = "https://domains.google.com/checkip"
		case Ifconfig:
			url = "https://ifconfig.io/ip"
		case Ipinfo:
			url = "https://ipinfo.io/ip"
		case Opendns:
			url = "https://diagnostic.opendns.com/myip"
		}
	}

	if len(url) == 0 {
		return "", false
	}

	return url, true
}
