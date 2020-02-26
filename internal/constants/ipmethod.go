package constants

import (
	"github.com/qdm12/ddns-updater/internal/models"
)

const (
	PROVIDER models.IPMethod = "provider"
	GOOGLE   models.IPMethod = "google"
	OPENDNS  models.IPMethod = "opendns"
	IFCONFIG models.IPMethod = "ifconfig"
	IPINFO   models.IPMethod = "ipinfo"
	CYCLE    models.IPMethod = "cycle"
)

func IPMethodMapping() map[models.IPMethod]string {
	return map[models.IPMethod]string{
		PROVIDER: string(PROVIDER),
		CYCLE:    string(CYCLE),
		GOOGLE:   "https://google.com/search?q=ip",
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
	choices = IPMethodChoices()
	for i, choice := range choices {
		if choice == CYCLE || choice == PROVIDER {
			choices[i] = choices[len(choices)-1]
			choices = choices[:len(choices)-1]
		}
	}
	return choices
}
