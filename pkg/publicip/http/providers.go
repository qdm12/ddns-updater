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
	Google   Provider = "google"
	Ifconfig Provider = "ifconfig"
	Ipify    Provider = "ipify"
	Ipinfo   Provider = "ipinfo"
	Noip     Provider = "noip"
)

func ListProviders() []Provider {
	return []Provider{
		Google,
		Ifconfig,
		Ipify,
		Ipinfo,
		Noip,
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
	ErrUnknownProvider   = errors.New("unknown provider")
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

func (provider Provider) url(version ipversion.IPVersion) (url string, ok bool) {
	switch version {
	case ipversion.IP4:
		switch provider { //nolint:exhaustive
		case Ipify:
			url = "https://api.ipify.org"
		case Noip:
			url = "http://ip1.dynupdate.no-ip.com"
		}

	case ipversion.IP6:
		switch provider { //nolint:exhaustive
		case Ipify:
			url = "https://api6.ipify.org"
		case Noip:
			url = "http://ip1.dynupdate6.no-ip.com"
		}

	case ipversion.IP4or6:
		switch provider { //nolint:exhaustive
		case Google:
			url = "https://domains.google.com/checkip"
		case Ifconfig:
			url = "https://ifconfig.io/ip"
		case Ipinfo:
			url = "https://ipinfo.io/ip"
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
