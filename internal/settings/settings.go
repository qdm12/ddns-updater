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
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/providers/cloudflare"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dd24"
	"github.com/qdm12/ddns-updater/internal/settings/providers/ddnss"
	"github.com/qdm12/ddns-updater/internal/settings/providers/digitalocean"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dnsomatic"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dnspod"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dondominio"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dreamhost"
	"github.com/qdm12/ddns-updater/internal/settings/providers/duckdns"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dyn"
	"github.com/qdm12/ddns-updater/internal/settings/providers/dynv6"
	"github.com/qdm12/ddns-updater/internal/settings/providers/freedns"
	"github.com/qdm12/ddns-updater/internal/settings/providers/gandi"
	"github.com/qdm12/ddns-updater/internal/settings/providers/godaddy"
	"github.com/qdm12/ddns-updater/internal/settings/providers/google"
	"github.com/qdm12/ddns-updater/internal/settings/providers/he"
	"github.com/qdm12/ddns-updater/internal/settings/providers/infomaniak"
	"github.com/qdm12/ddns-updater/internal/settings/providers/linode"
	"github.com/qdm12/ddns-updater/internal/settings/providers/luadns"
	"github.com/qdm12/ddns-updater/internal/settings/providers/namecheap"
	"github.com/qdm12/ddns-updater/internal/settings/providers/njalla"
	"github.com/qdm12/ddns-updater/internal/settings/providers/noip"
	"github.com/qdm12/ddns-updater/internal/settings/providers/opendns"
	"github.com/qdm12/ddns-updater/internal/settings/providers/ovh"
	"github.com/qdm12/ddns-updater/internal/settings/providers/porkbun"
	"github.com/qdm12/ddns-updater/internal/settings/providers/selfhostde"
	"github.com/qdm12/ddns-updater/internal/settings/providers/servercow"
	"github.com/qdm12/ddns-updater/internal/settings/providers/spdyn"
	"github.com/qdm12/ddns-updater/internal/settings/providers/strato"
	"github.com/qdm12/ddns-updater/internal/settings/providers/variomedia"
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
	ipVersion ipversion.IPVersion, matcher regex.Matcher, logger log.Logger) (
	settings Settings, err error) {
	switch provider {
	case constants.Cloudflare:
		return cloudflare.New(data, domain, host, ipVersion, matcher, logger)
	case constants.Dd24:
		return dd24.New(data, domain, host, ipVersion, logger)
	case constants.DdnssDe:
		return ddnss.New(data, domain, host, ipVersion, logger)
	case constants.DigitalOcean:
		return digitalocean.New(data, domain, host, ipVersion, logger)
	case constants.DnsOMatic:
		return dnsomatic.New(data, domain, host, ipVersion, matcher, logger)
	case constants.DNSPod:
		return dnspod.New(data, domain, host, ipVersion, logger)
	case constants.DonDominio:
		return dondominio.New(data, domain, host, ipVersion, logger)
	case constants.Dreamhost:
		return dreamhost.New(data, domain, host, ipVersion, matcher, logger)
	case constants.DuckDNS:
		return duckdns.New(data, domain, host, ipVersion, matcher, logger)
	case constants.Dyn:
		return dyn.New(data, domain, host, ipVersion, logger)
	case constants.DynV6:
		return dynv6.New(data, domain, host, ipVersion, logger)
	case constants.FreeDNS:
		return freedns.New(data, domain, host, ipVersion, logger)
	case constants.Gandi:
		return gandi.New(data, domain, host, ipVersion, logger)
	case constants.GoDaddy:
		return godaddy.New(data, domain, host, ipVersion, matcher, logger)
	case constants.Google:
		return google.New(data, domain, host, ipVersion, logger)
	case constants.HE:
		return he.New(data, domain, host, ipVersion, logger)
	case constants.Infomaniak:
		return infomaniak.New(data, domain, host, ipVersion, logger)
	case constants.Linode:
		return linode.New(data, domain, host, ipVersion, logger)
	case constants.LuaDNS:
		return luadns.New(data, domain, host, ipVersion, logger)
	case constants.Namecheap:
		return namecheap.New(data, domain, host, ipVersion, matcher, logger)
	case constants.Njalla:
		return njalla.New(data, domain, host, ipVersion, logger)
	case constants.NoIP:
		return noip.New(data, domain, host, ipVersion, logger)
	case constants.OpenDNS:
		return opendns.New(data, domain, host, ipVersion, logger)
	case constants.OVH:
		return ovh.New(data, domain, host, ipVersion, logger)
	case constants.Porkbun:
		return porkbun.New(data, domain, host, ipVersion, logger)
	case constants.SelfhostDe:
		return selfhostde.New(data, domain, host, ipVersion, logger)
	case constants.Servercow:
		return servercow.New(data, domain, host, ipVersion, logger)
	case constants.Spdyn:
		return spdyn.New(data, domain, host, ipVersion, logger)
	case constants.Strato:
		return strato.New(data, domain, host, ipVersion, logger)
	case constants.Variomedia:
		return variomedia.New(data, domain, host, ipVersion, matcher, logger)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderUnknown, provider)
	}
}
