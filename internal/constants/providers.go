package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values.
const (
	CLOUDFLARE   models.Provider = "cloudflare"
	DIGITALOCEAN models.Provider = "digitalocean"
	DDNSSDE      models.Provider = "ddnss"
	DONDOMINIO   models.Provider = "dondominio"
	DNSOMATIC    models.Provider = "dnsomatic"
	DNSPOD       models.Provider = "dnspod"
	DUCKDNS      models.Provider = "duckdns"
	DYN          models.Provider = "dyn"
	VARIOMEDIA   models.Provider = "variomedia"
	DYNV6        models.Provider = "dynv6"
	DREAMHOST    models.Provider = "dreamhost"
	FREEDNS      models.Provider = "freedns"
	GANDI        models.Provider = "gandi"
	GODADDY      models.Provider = "godaddy"
	GOOGLE       models.Provider = "google"
	HE           models.Provider = "he"
	INFOMANIAK   models.Provider = "infomaniak"
	LINODE       models.Provider = "linode"
	LUADNS       models.Provider = "luadns"
	NAMECHEAP    models.Provider = "namecheap"
	NOIP         models.Provider = "noip"
	OPENDNS      models.Provider = "opendns"
	OVH          models.Provider = "ovh"
	SELFHOSTDE   models.Provider = "selfhost.de"
	STRATO       models.Provider = "strato"
)

func ProviderChoices() []models.Provider {
	return []models.Provider{
		CLOUDFLARE,
		DIGITALOCEAN,
		DDNSSDE,
		DONDOMINIO,
		DNSOMATIC,
		DNSPOD,
		DUCKDNS,
		DYN,
		VARIOMEDIA,
		DYNV6,
		DREAMHOST,
		FREEDNS,
		GANDI,
		GODADDY,
		GOOGLE,
		HE,
		INFOMANIAK,
		LINODE,
		LUADNS,
		NAMECHEAP,
		NOIP,
		OVH,
		OPENDNS,
		SELFHOSTDE,
		STRATO,
	}
}
