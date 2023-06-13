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
	OpenDNS    Provider = "opendns"
)

func ListProviders() []Provider {
	return []Provider{
		Cloudflare,
		Google,
		OpenDNS,
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
	qType      dns.Type
}

func (provider Provider) data() providerData {
	switch provider {
	case Google:
		return providerData{
			nameserver: "ns1.google.com:53",
			fqdn:       "o-o.myaddr.l.google.com.",
			class:      dns.ClassINET,
			qType:      dns.Type(dns.TypeTXT),
		}
	case Cloudflare:
		return providerData{
			nameserver: "one.one.one.one:53",
			fqdn:       "whoami.cloudflare.",
			class:      dns.ClassCHAOS,
			qType:      dns.Type(dns.TypeTXT),
		}
	case OpenDNS:
		return providerData{
			nameserver: "resolver1.opendns.com:53",
			fqdn:       "myip.opendns.com.",
			class:      dns.ClassINET,
			qType:      dns.Type(dns.TypeANY),
		}
	}
	panic(`provider unknown: "` + string(provider) + `"`)
}
