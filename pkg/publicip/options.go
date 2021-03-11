package publicip

import (
	"net/http"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	iphttp "github.com/qdm12/ddns-updater/pkg/publicip/http"
)

type settings struct {
	// If both dns and http are enabled it will cycle between both of them.
	dns  dnsSettings
	http httpSettings
}

type dnsSettings struct {
	enabled bool
	options []dns.Option
}

type httpSettings struct {
	enabled bool
	client  *http.Client
	options []iphttp.Option
}

func defaultSettings() settings {
	return settings{
		dns: dnsSettings{
			enabled: true,
			options: []dns.Option{},
		},
		http: httpSettings{
			enabled: false,
			client:  &http.Client{},
			options: []iphttp.Option{},
		},
	}
}

type Option func(s *settings) error

func UseDNS(options ...dns.Option) Option {
	return func(s *settings) error {
		s.dns.enabled = true
		s.dns.options = options
		return nil
	}
}

func UseHTTP(client *http.Client, options ...iphttp.Option) Option {
	return func(s *settings) error {
		s.http.enabled = true
		s.http.client = client
		s.http.options = options
		return nil
	}
}
