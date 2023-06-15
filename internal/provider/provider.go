package provider

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/providers/aliyun"
	"github.com/qdm12/ddns-updater/internal/provider/providers/allinkl"
	"github.com/qdm12/ddns-updater/internal/provider/providers/cloudflare"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dd24"
	"github.com/qdm12/ddns-updater/internal/provider/providers/ddnss"
	"github.com/qdm12/ddns-updater/internal/provider/providers/digitalocean"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dnsomatic"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dnspod"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dondominio"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dreamhost"
	"github.com/qdm12/ddns-updater/internal/provider/providers/duckdns"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dyn"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dynu"
	"github.com/qdm12/ddns-updater/internal/provider/providers/dynv6"
	"github.com/qdm12/ddns-updater/internal/provider/providers/easydns"
	"github.com/qdm12/ddns-updater/internal/provider/providers/freedns"
	"github.com/qdm12/ddns-updater/internal/provider/providers/gandi"
	"github.com/qdm12/ddns-updater/internal/provider/providers/gcp"
	"github.com/qdm12/ddns-updater/internal/provider/providers/godaddy"
	"github.com/qdm12/ddns-updater/internal/provider/providers/google"
	"github.com/qdm12/ddns-updater/internal/provider/providers/he"
	"github.com/qdm12/ddns-updater/internal/provider/providers/infomaniak"
	"github.com/qdm12/ddns-updater/internal/provider/providers/inwx"
	"github.com/qdm12/ddns-updater/internal/provider/providers/linode"
	"github.com/qdm12/ddns-updater/internal/provider/providers/luadns"
	"github.com/qdm12/ddns-updater/internal/provider/providers/namecheap"
	"github.com/qdm12/ddns-updater/internal/provider/providers/namecom"
	"github.com/qdm12/ddns-updater/internal/provider/providers/netcup"
	"github.com/qdm12/ddns-updater/internal/provider/providers/njalla"
	"github.com/qdm12/ddns-updater/internal/provider/providers/noip"
	"github.com/qdm12/ddns-updater/internal/provider/providers/opendns"
	"github.com/qdm12/ddns-updater/internal/provider/providers/ovh"
	"github.com/qdm12/ddns-updater/internal/provider/providers/porkbun"
	"github.com/qdm12/ddns-updater/internal/provider/providers/selfhostde"
	"github.com/qdm12/ddns-updater/internal/provider/providers/servercow"
	"github.com/qdm12/ddns-updater/internal/provider/providers/spdyn"
	"github.com/qdm12/ddns-updater/internal/provider/providers/strato"
	"github.com/qdm12/ddns-updater/internal/provider/providers/variomedia"
	"github.com/qdm12/ddns-updater/internal/provider/providers/zoneedit"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider interface {
	String() string
	Domain() string
	Host() string
	BuildDomainName() string
	HTML() models.HTMLRow
	Proxied() bool
	IPVersion() ipversion.IPVersion
	Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error)
}

var ErrProviderUnknown = errors.New("unknown provider")

//nolint:gocyclo
func New(providerName models.Provider, data json.RawMessage, domain, host string, //nolint:ireturn
	ipVersion ipversion.IPVersion) (provider Provider, err error) {
	switch providerName {
	case constants.Aliyun:
		return aliyun.New(data, domain, host, ipVersion)
	case constants.AllInkl:
		return allinkl.New(data, domain, host, ipVersion)
	case constants.Cloudflare:
		return cloudflare.New(data, domain, host, ipVersion)
	case constants.Dd24:
		return dd24.New(data, domain, host, ipVersion)
	case constants.DdnssDe:
		return ddnss.New(data, domain, host, ipVersion)
	case constants.DigitalOcean:
		return digitalocean.New(data, domain, host, ipVersion)
	case constants.DNSOMatic:
		return dnsomatic.New(data, domain, host, ipVersion)
	case constants.DNSPod:
		return dnspod.New(data, domain, host, ipVersion)
	case constants.DonDominio:
		return dondominio.New(data, domain, host, ipVersion)
	case constants.Dreamhost:
		return dreamhost.New(data, domain, host, ipVersion)
	case constants.DuckDNS:
		return duckdns.New(data, domain, host, ipVersion)
	case constants.Dyn:
		return dyn.New(data, domain, host, ipVersion)
	case constants.Dynu:
		return dynu.New(data, domain, host, ipVersion)
	case constants.DynV6:
		return dynv6.New(data, domain, host, ipVersion)
	case constants.EasyDNS:
		return easydns.New(data, domain, host, ipVersion)
	case constants.FreeDNS:
		return freedns.New(data, domain, host, ipVersion)
	case constants.Gandi:
		return gandi.New(data, domain, host, ipVersion)
	case constants.GCP:
		return gcp.New(data, domain, host, ipVersion)
	case constants.GoDaddy:
		return godaddy.New(data, domain, host, ipVersion)
	case constants.Google:
		return google.New(data, domain, host, ipVersion)
	case constants.HE:
		return he.New(data, domain, host, ipVersion)
	case constants.Infomaniak:
		return infomaniak.New(data, domain, host, ipVersion)
	case constants.INWX:
		return inwx.New(data, domain, host, ipVersion)
	case constants.Linode:
		return linode.New(data, domain, host, ipVersion)
	case constants.LuaDNS:
		return luadns.New(data, domain, host, ipVersion)
	case constants.Namecheap:
		return namecheap.New(data, domain, host, ipVersion)
	case constants.NameCom:
		return namecom.New(data, domain, host, ipVersion)
	case constants.Netcup:
		return netcup.New(data, domain, host, ipVersion)
	case constants.Njalla:
		return njalla.New(data, domain, host, ipVersion)
	case constants.NoIP:
		return noip.New(data, domain, host, ipVersion)
	case constants.OpenDNS:
		return opendns.New(data, domain, host, ipVersion)
	case constants.OVH:
		return ovh.New(data, domain, host, ipVersion)
	case constants.Porkbun:
		return porkbun.New(data, domain, host, ipVersion)
	case constants.SelfhostDe:
		return selfhostde.New(data, domain, host, ipVersion)
	case constants.Servercow:
		return servercow.New(data, domain, host, ipVersion)
	case constants.Spdyn:
		return spdyn.New(data, domain, host, ipVersion)
	case constants.Strato:
		return strato.New(data, domain, host, ipVersion)
	case constants.Variomedia:
		return variomedia.New(data, domain, host, ipVersion)
	case constants.Zoneedit:
		return zoneedit.New(data, domain, host, ipVersion)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProviderUnknown, providerName)
	}
}
