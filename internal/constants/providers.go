package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values
const (
	CLOUDFLARE models.Provider = "cloudflare"
	DDNSSDE    models.Provider = "ddnss"
	DNSPOD     models.Provider = "dnspod"
	DUCKDNS    models.Provider = "duckdns"
	DYN        models.Provider = "dyn"
	DREAMHOST  models.Provider = "dreamhost"
	GODADDY    models.Provider = "godaddy"
	GOOGLE     models.Provider = "google"
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
		DYN,
		DREAMHOST,
		GODADDY,
		GOOGLE,
		INFOMANIAK,
		NAMECHEAP,
		NOIP,
	}
}
