package settings

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Settings interface {
	String() string
	Domain() string
	Host() string
	BuildDomainName() string
	HTML() models.HTMLRow
	Proxied() bool
	IPVersion() ipversion.IPVersion
	Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error)
}

var ErrProviderUnknown = errors.New("unknown provider")

func New(provider models.Provider, data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, matcher regex.Matcher) (settings Settings, err error) {
	switch provider {
	case constants.Cloudflare:
		return NewCloudflare(data, domain, host, ipVersion, matcher)
	case constants.DdnssDe:
		return NewDdnss(data, domain, host, ipVersion)
	case constants.DigitalOcean:
		return NewDigitalOcean(data, domain, host, ipVersion)
	case constants.DnsOMatic:
		return NewDNSOMatic(data, domain, host, ipVersion, matcher)
	case constants.DNSPod:
		return NewDNSPod(data, domain, host, ipVersion)
	case constants.DonDominio:
		return NewDonDominio(data, domain, host, ipVersion)
	case constants.Dreamhost:
		return NewDreamhost(data, domain, host, ipVersion, matcher)
	case constants.DuckDNS:
		return NewDuckdns(data, domain, host, ipVersion, matcher)
	case constants.Dyn:
		return NewDyn(data, domain, host, ipVersion)
	case constants.DynV6:
		return NewDynV6(data, domain, host, ipVersion)
	case constants.FreeDNS:
		return NewFreedns(data, domain, host, ipVersion)
	case constants.Gandi:
		return NewGandi(data, domain, host, ipVersion)
	case constants.GoDaddy:
		return NewGodaddy(data, domain, host, ipVersion, matcher)
	case constants.Google:
		return NewGoogle(data, domain, host, ipVersion)
	case constants.HE:
		return NewHe(data, domain, host, ipVersion)
	case constants.Infomaniak:
		return NewInfomaniak(data, domain, host, ipVersion)
	case constants.Linode:
		return NewLinode(data, domain, host, ipVersion)
	case constants.LuaDNS:
		return NewLuaDNS(data, domain, host, ipVersion)
	case constants.Namecheap:
		return NewNamecheap(data, domain, host, ipVersion, matcher)
	case constants.Njalla:
		return NewNjalla(data, domain, host, ipVersion)
	case constants.NoIP:
		return NewNoip(data, domain, host, ipVersion)
	case constants.OpenDNS:
		return NewOpendns(data, domain, host, ipVersion)
	case constants.OVH:
		return NewOVH(data, domain, host, ipVersion)
	case constants.SelfhostDe:
		return NewSelfhostde(data, domain, host, ipVersion)
	case constants.Spdyn:
		return NewSpdyn(data, domain, host, ipVersion)
	case constants.Strato:
		return NewStrato(data, domain, host, ipVersion)
	case constants.Variomedia:
		return NewVariomedia(data, domain, host, ipVersion, matcher)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderUnknown, provider)
	}
}
