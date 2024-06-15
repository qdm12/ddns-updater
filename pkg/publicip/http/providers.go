package http

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider string

const (
	Google    Provider = "google"
	Ifconfig  Provider = "ifconfig"
	Ipify     Provider = "ipify"
	Ipinfo    Provider = "ipinfo"
	Spdyn     Provider = "spdyn"
	Ipleak    Provider = "ipleak"
	Icanhazip Provider = "icanhazip"
	Ident     Provider = "ident"
	Nnev      Provider = "nnev"
	Wtfismyip Provider = "wtfismyip"
	Seeip     Provider = "seeip"
	Changeip  Provider = "changeip"
)

func ListProviders() []Provider {
	return []Provider{
		Google,
		Ifconfig,
		Ipify,
		Ipinfo,
		Spdyn,
		Ipleak,
		Icanhazip,
		Ident,
		Nnev,
		Wtfismyip,
		Seeip,
		Changeip,
	}
}

func ListProvidersForVersion(version ipversion.IPVersion) (providers []Provider) {
	allProviders := ListProviders()
	for _, provider := range allProviders {
		if provider.SupportsVersion(version) {
			providers = append(providers, provider)
		}
	}
	return providers
}

var (
	ErrUnknownProvider   = errors.New("unknown public IP echo HTTP provider")
	ErrProviderIPVersion = errors.New("provider does not support IP version")
)

func ValidateProvider(provider Provider, version ipversion.IPVersion) error {
	if strings.HasPrefix(string(provider), "url:https://") { // custom HTTP url
		return nil
	}

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

func (provider Provider) url(version ipversion.IPVersion) (url string, ok bool) { //nolint:gocyclo
	switch version {
	case ipversion.IP4:
		switch provider { //nolint:exhaustive
		case Ipify:
			url = "https://api.ipify.org"
		case Ipleak:
			url = "https://ipv4.ipleak.net/json"
		case Icanhazip:
			url = "https://ipv4.icanhazip.com"
		case Ident:
			url = "https://v4.ident.me"
		case Nnev:
			url = "https://ip4.nnev.de"
		case Wtfismyip:
			url = "https://ipv4.wtfismyip.com/text"
		case Seeip:
			url = "https://ipv4.seeip.org"
		}

	case ipversion.IP6:
		switch provider { //nolint:exhaustive
		case Ipify:
			url = "https://api6.ipify.org"
		case Ipleak:
			url = "https://ipv6.ipleak.net/json"
		case Icanhazip:
			url = "https://ipv6.icanhazip.com"
		case Ident:
			url = "https://v6.ident.me"
		case Nnev:
			url = "https://ip6.nnev.de"
		case Wtfismyip:
			url = "https://ipv6.wtfismyip.com/text"
		case Seeip:
			url = "https://ipv6.seeip.org"
		}

	case ipversion.IP4or6:
		switch provider {
		case Ipify:
			url = "https://api64.ipify.org"
		case Google:
			url = "https://domains.google.com/checkip"
		case Ifconfig:
			url = "https://ifconfig.io/ip"
		case Ipinfo:
			url = "https://ipinfo.io/ip"
		case Spdyn:
			url = "https://checkip.spdyn.de"
		case Ipleak:
			url = "https://ipleak.net/json"
		case Icanhazip:
			url = "https://icanhazip.com"
		case Ident:
			url = "https://ident.me"
		case Nnev:
			url = "https://ip.nnev.de"
		case Wtfismyip:
			url = "https://wtfismyip.com/text"
		case Seeip:
			url = "https://api.seeip.org"
		case Changeip:
			url = "https://ip.changeip.com"
		}
	}

	// Custom URL?
	if s := string(provider); strings.HasPrefix(s, "url:") {
		url = strings.TrimPrefix(s, "url:")
	}

	if url == "" {
		return "", false
	}

	return url, true
}

func (provider Provider) SupportsVersion(version ipversion.IPVersion) bool {
	_, ok := provider.url(version)
	return ok
}

// CustomProvider creates a provider with a custom HTTP(s) URL.
// It is the responsibility of the caller to make sure it is a valid URL
// and that it supports the desired IP version(s) as no further check is
// done on it.
func CustomProvider(httpsURL *url.URL) Provider { //nolint:interfacer
	return Provider("url:" + httpsURL.String())
}
