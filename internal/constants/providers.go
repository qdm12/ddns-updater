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
	DYNV6        models.Provider = "dynv6"
	DREAMHOST    models.Provider = "dreamhost"
	GODADDY      models.Provider = "godaddy"
	GOOGLE       models.Provider = "google"
	HE           models.Provider = "he"
	INFOMANIAK   models.Provider = "infomaniak"
	LUADNS       models.Provider = "luadns"
	NAMECHEAP    models.Provider = "namecheap"
	NOIP         models.Provider = "noip"
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
		DYNV6,
		DREAMHOST,
		GODADDY,
		GOOGLE,
		HE,
		INFOMANIAK,
		LUADNS,
		NAMECHEAP,
		NOIP,
		OVH,
		SELFHOSTDE,
		STRATO,
	}
}
