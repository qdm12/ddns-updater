package constants

import (
	"github.com/qdm12/ddns-updater/internal/models"
)

const (
	PROVIDER models.IPMethod = "provider"
	OPENDNS  models.IPMethod = "opendns"
	IFCONFIG models.IPMethod = "ifconfig"
	IPINFO   models.IPMethod = "ipinfo"
	IPIFY    models.IPMethod = "ipify"
	IPIFY6   models.IPMethod = "ipify6"
	CYCLE    models.IPMethod = "cycle"
	DDNSS    models.IPMethod = "ddnss"
	DDNSS4   models.IPMethod = "ddnss4"
	DDNSS6   models.IPMethod = "ddnss6"
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
		IPIFY:    "https://api.ipify.org",
		IPIFY6:   "https://api6.ipify.org",
		DDNSS:    "https://ip4.ddnss.de/meineip.php",
		DDNSS4:   "https://ip4.ddnss.de/meineip.php",
		DDNSS6:   "https://ip6.ddnss.de/meineip.php",
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
		switch choice {
		case PROVIDER, CYCLE:
		default:
			choices = append(choices, choice)
		}
	}
	return choices
}
