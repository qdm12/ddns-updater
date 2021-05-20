package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values.
const (
	Cloudflare   models.Provider = "cloudflare"
	DdnssDe      models.Provider = "ddnss"
	DigitalOcean models.Provider = "digitalocean"
	DnsOMatic    models.Provider = "dnsomatic"
	DNSPod       models.Provider = "dnspod"
	DonDominio   models.Provider = "dondominio"
	Dreamhost    models.Provider = "dreamhost"
	DuckDNS      models.Provider = "duckdns"
	Dyn          models.Provider = "dyn"
	DynV6        models.Provider = "dynv6"
	FreeDNS      models.Provider = "freedns"
	Gandi        models.Provider = "gandi"
	GoDaddy      models.Provider = "godaddy"
	Google       models.Provider = "google"
	HE           models.Provider = "he"
	Infomaniak   models.Provider = "infomaniak"
	Linode       models.Provider = "linode"
	LuaDNS       models.Provider = "luadns"
	Namecheap    models.Provider = "namecheap"
	Njalla       models.Provider = "njalla"
	NoIP         models.Provider = "noip"
	OpenDNS      models.Provider = "opendns"
	OVH          models.Provider = "ovh"
	SelfhostDe   models.Provider = "selfhost.de"
	Spdyn        models.Provider = "spdyn"
	Strato       models.Provider = "strato"
	Variomedia   models.Provider = "variomedia"
)

func ProviderChoices() []models.Provider {
	return []models.Provider{
		Cloudflare,
		DdnssDe,
		DigitalOcean,
		DnsOMatic,
		DNSPod,
		DonDominio,
		Dreamhost,
		DuckDNS,
		Dyn,
		DynV6,
		FreeDNS,
		Gandi,
		GoDaddy,
		Google,
		HE,
		Infomaniak,
		Linode,
		LuaDNS,
		Namecheap,
		Njalla,
		NoIP,
		OpenDNS,
		OVH,
		SelfhostDe,
		Spdyn,
		Strato,
		Variomedia,
	}
}
