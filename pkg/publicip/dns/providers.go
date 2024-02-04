package dns

import (
	"errors"
	"fmt"
	"net/netip"

	"github.com/miekg/dns"
)

type Provider string

const (
	Cloudflare Provider = "cloudflare"
	OpenDNS    Provider = "opendns"
)

func ListProviders() []Provider {
	return []Provider{
		Cloudflare,
		OpenDNS,
	}
}

var ErrUnknownProvider = errors.New("unknown public IP echo DNS provider")

func ValidateProvider(provider Provider) error {
	for _, possible := range ListProviders() {
		if provider == possible {
			return nil
		}
	}
	return fmt.Errorf("%w: %s", ErrUnknownProvider, provider)
}

type providerData struct {
	// Address for IPv4 or IPv6.
	Address string
	IPv4    netip.Addr
	IPv6    netip.Addr
	TLSName string
	fqdn    string
	class   dns.Class
	qType   dns.Type
}

func (p Provider) data() providerData {
	switch p {
	// Note on deprecating Google:
	// Only their nameserver ns1.google.com returns your public IP address.
	// All their other nameservers return the closest Google datacenter IP.
	// Unfortunately, ns1.google.com is not compatible with DNS over TLS,
	// and dns.google.com is but does not echo your IP address.
	// dig TXT @ns1.google.com o-o.myaddr.l.google.com +tls
	// dig TXT @dns.google.com o-o.myaddr.l.google.com +tls
	case Cloudflare:
		return providerData{
			Address: "1dot1dot1dot1.cloudflare-dns.com",
			IPv4:    netip.AddrFrom4([4]byte{1, 1, 1, 1}),
			IPv6:    netip.AddrFrom16([16]byte{0x26, 0x6, 0x47, 0x0, 0x47, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x11, 0x11}), //nolint:lll
			TLSName: "cloudflare-dns.com",
			fqdn:    "whoami.cloudflare.",
			class:   dns.ClassCHAOS,
			qType:   dns.Type(dns.TypeTXT),
		}
	case OpenDNS:
		return providerData{
			Address: "dns.opendns.com",
			IPv4:    netip.AddrFrom4([4]byte{208, 67, 222, 222}),
			IPv6:    netip.AddrFrom16([16]byte{0x26, 0x20, 0x1, 0x19, 0x0, 0x35, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x35}), //nolint:lll
			TLSName: "dns.opendns.com",
			fqdn:    "myip.opendns.com.",
			class:   dns.ClassINET,
			qType:   dns.Type(dns.TypeANY),
		}
	}
	panic(`provider unknown: "` + string(p) + `"`)
}
