package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values.
const (
	Aliyun       models.Provider = "aliyun"
	AllInkl      models.Provider = "allinkl"
	Changeip     models.Provider = "changeip"
	Cloudflare   models.Provider = "cloudflare"
	Custom       models.Provider = "custom"
	Dd24         models.Provider = "dd24"
	DdnssDe      models.Provider = "ddnss"
	DeSEC        models.Provider = "desec"
	DigitalOcean models.Provider = "digitalocean"
	DNSOMatic    models.Provider = "dnsomatic"
	DNSPod       models.Provider = "dnspod"
	DonDominio   models.Provider = "dondominio"
	Dreamhost    models.Provider = "dreamhost"
	DuckDNS      models.Provider = "duckdns"
	Dyn          models.Provider = "dyn"
	Dynu         models.Provider = "dynu"
	DynV6        models.Provider = "dynv6"
	EasyDNS      models.Provider = "easydns"
	Example      models.Provider = "example"
	FreeDNS      models.Provider = "freedns"
	Gandi        models.Provider = "gandi"
	GCP          models.Provider = "gcp"
	GoDaddy      models.Provider = "godaddy"
	GoIP         models.Provider = "goip"
	HE           models.Provider = "he"
	Hetzner      models.Provider = "hetzner"
	Infomaniak   models.Provider = "infomaniak"
	INWX         models.Provider = "inwx"
	Ionos        models.Provider = "ionos"
	Linode       models.Provider = "linode"
	LuaDNS       models.Provider = "luadns"
	Namecheap    models.Provider = "namecheap"
	NameCom      models.Provider = "name.com"
	Netcup       models.Provider = "netcup"
	Njalla       models.Provider = "njalla"
	NoIP         models.Provider = "noip"
	NowDNS       models.Provider = "nowdns"
	OpenDNS      models.Provider = "opendns"
	OVH          models.Provider = "ovh"
	Porkbun      models.Provider = "porkbun"
	Route53      models.Provider = "route53"
	SelfhostDe   models.Provider = "selfhost.de"
	Servercow    models.Provider = "servercow"
	Spdyn        models.Provider = "spdyn"
	Strato       models.Provider = "strato"
	Variomedia   models.Provider = "variomedia"
	Zoneedit     models.Provider = "zoneedit"
)

func ProviderChoices() []models.Provider {
	return []models.Provider{
		Aliyun,
		AllInkl,
		Changeip,
		Cloudflare,
		Dd24,
		DdnssDe,
		DeSEC,
		DigitalOcean,
		DNSOMatic,
		DNSPod,
		DonDominio,
		Dreamhost,
		DuckDNS,
		Dyn,
		Dynu,
		DynV6,
		EasyDNS,
		Example,
		FreeDNS,
		Gandi,
		GCP,
		GoDaddy,
		GoIP,
		HE,
		Hetzner,
		Infomaniak,
		INWX,
		Ionos,
		Linode,
		LuaDNS,
		Namecheap,
		NameCom,
		Njalla,
		NoIP,
		NowDNS,
		OpenDNS,
		OVH,
		Porkbun,
		Route53,
		SelfhostDe,
		Spdyn,
		Strato,
		Variomedia,
		Zoneedit,
	}
}
