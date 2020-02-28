package constants

import (
	"github.com/qdm12/ddns-updater/internal/models"
)

const (
	PROVIDER models.IPMethod = "provider"
	OPENDNS  models.IPMethod = "opendns"
	IFCONFIG models.IPMethod = "ifconfig"
	IPINFO   models.IPMethod = "ipinfo"
	CYCLE    models.IPMethod = "cycle"
	// Retro compatibility only
	GOOGLE models.IPMethod = "google"
)

func IPMethodMapping() map[models.IPMethod]string {
	return map[models.IPMethod]string{
		PROVIDER: string(PROVIDER),
		CYCLE:    string(CYCLE),
		OPENDNS:  "https://diagnostic.opendns.com/myip",
		IFCONFIG: "https://ifconfig.io/ip",
		IPINFO:   "https://ipinfo.io/ip",
	}
}

func IPMethodChoices() (choices []models.IPMethod) {
	for choice := range IPMethodMapping() {
		choices = append(choices, choice)
	}
	return choices
}

func IPMethodExternalChoices() (choices []models.IPMethod) {
	for _, choice := range IPMethodChoices() {
		if choice != CYCLE && choice != PROVIDER {
			choices = append(choices, choice)
		}
	}
	return choices
}
