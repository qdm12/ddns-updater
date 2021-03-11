package dns

import (
	"errors"
	"fmt"

	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider string

const (
	Cloudflare Provider = "cloudflare"
	Google     Provider = "google"
)

func ListProviders() []Provider {
	return []Provider{
		Cloudflare,
		Google,
	}
}

var ErrUnknownProvider = errors.New("unknown provider")

func ValidateProvider(provider Provider) error {
	for _, possible := range ListProviders() {
		if provider == possible {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
}

func getProviderData(provider Provider, v ipversion.IPVersion) (nameserver, txtRecord string) {
	switch provider {
	case Google:
		switch v {
		case ipversion.IP4:
			return "ns1.google.com:53", "o-o.myaddr.l.google.com"
		case ipversion.IP6:
			return "ns2.google.com:53", "o-o.myaddr.l.google.com"
		default:
			return "ns1.google.com:53", "o-o.myaddr.l.google.com"
		}
	case Cloudflare:
		return "one.one.one.one:53", "whoami.cloudflare"
	}
	panic(`combination not set for provider "` +
		string(provider) + `" and ip version "` + v.String() + `"`)
}
