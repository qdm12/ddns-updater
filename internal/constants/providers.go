package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values
const (
	CLOUDFLARE models.Provider = "cloudflare"
	DDNSSDE    models.Provider = "ddnss"
	DONDOMINIO models.Provider = "dondominio"
	DNSPOD     models.Provider = "dnspod"
	DUCKDNS    models.Provider = "duckdns"
	DYN        models.Provider = "dyn"
	DREAMHOST  models.Provider = "dreamhost"
	GODADDY    models.Provider = "godaddy"
	GOOGLE     models.Provider = "google"
	HE         models.Provider = "he"
	INFOMANIAK models.Provider = "infomaniak"
	NAMECHEAP  models.Provider = "namecheap"
	NOIP       models.Provider = "noip"
)

func ProviderChoices() []models.Provider {
	return []models.Provider{
		CLOUDFLARE,
		DDNSSDE,
		DONDOMINIO,
		DNSPOD,
		DUCKDNS,
		DYN,
		DREAMHOST,
		GODADDY,
		GOOGLE,
		HE,
		INFOMANIAK,
		NAMECHEAP,
		NOIP,
	}
}
