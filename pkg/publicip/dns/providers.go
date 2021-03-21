package dns

import (
	"errors"
	"fmt"

	"github.com/miekg/dns"
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

type providerData struct {
	nameserver string
	fqdn       string
	class      dns.Class
}

func (provider Provider) data() providerData {
	switch provider {
	case Google:
		return providerData{
			nameserver: "ns1.google.com:53",
			fqdn:       "o-o.myaddr.l.google.com.",
			class:      dns.ClassINET,
		}
	case Cloudflare:
		return providerData{
			nameserver: "one.one.one.one:53",
			fqdn:       "whoami.cloudflare.",
			class:      dns.ClassCHAOS,
		}
	}
	panic(`provider unknown: "` + string(provider) + `"`)
}
