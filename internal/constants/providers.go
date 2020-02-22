package constants

import "github.com/qdm12/ddns-updater/internal/models"

// All possible provider values
const (
	PROVIDERGODADDY    models.Provider = "godaddy"
	PROVIDERNAMECHEAP  models.Provider = "namecheap"
	PROVIDERDUCKDNS    models.Provider = "duckdns"
	PROVIDERDREAMHOST  models.Provider = "dreamhost"
	PROVIDERCLOUDFLARE models.Provider = "cloudflare"
	PROVIDERNOIP       models.Provider = "noip"
	PROVIDERDNSPOD     models.Provider = "dnspod"
)

func ProviderChoices() (choices []models.Provider) {
	return []models.Provider{
		PROVIDERGODADDY,
		PROVIDERNAMECHEAP,
		PROVIDERDUCKDNS,
		PROVIDERDREAMHOST,
		PROVIDERCLOUDFLARE,
		PROVIDERNOIP,
		PROVIDERDNSPOD,
	}
}
