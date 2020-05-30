package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values
const (
	CLOUDFLARE models.Provider = "cloudflare"
	DDNSSDE    models.Provider = "ddnss"
	DNSPOD     models.Provider = "dnspod"
	DUCKDNS    models.Provider = "duckdns"
	DREAMHOST  models.Provider = "dreamhost"
	GODADDY    models.Provider = "godaddy"
	INFOMANIAK models.Provider = "infomaniak"
	NAMECHEAP  models.Provider = "namecheap"
	NOIP       models.Provider = "noip"
)

func ProviderChoices() []models.Provider {
	return []models.Provider{
		CLOUDFLARE,
		DDNSSDE,
		DNSPOD,
		DUCKDNS,
		DREAMHOST,
		GODADDY,
		INFOMANIAK,
		NAMECHEAP,
		NOIP,
	}
}
