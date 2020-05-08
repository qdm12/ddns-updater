package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values
const (
	GODADDY    models.Provider = "godaddy"
	NAMECHEAP  models.Provider = "namecheap"
	DUCKDNS    models.Provider = "duckdns"
	DREAMHOST  models.Provider = "dreamhost"
	CLOUDFLARE models.Provider = "cloudflare"
	NOIP       models.Provider = "noip"
	DNSPOD     models.Provider = "dnspod"
	INFOMANIAK models.Provider = "infomaniak"
	DDNSSDE    models.Provider = "ddnss"
	DYN        models.Provider = "dyn"
)

func ProviderChoices() []models.Provider {
	return []models.Provider{
		GODADDY,
		NAMECHEAP,
		DUCKDNS,
		DREAMHOST,
		CLOUDFLARE,
		NOIP,
		DNSPOD,
		INFOMANIAK,
		DDNSSDE,
	}
}
