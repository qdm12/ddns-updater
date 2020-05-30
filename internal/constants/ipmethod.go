package constants

import (
	"github.com/qdm12/ddns-updater/internal/models"
)

func IPMethods() []models.IPMethod {
	return []models.IPMethod{
		{
			Name: "cycle",
		},
		{
			Name: "opendns",
			URL:  "https://diagnostic.opendns.com/myip",
			IPv4: true,
			IPv6: true,
		},
		{
			Name: "ifconfig",
			URL:  "https://ifconfig.io/ip",
			IPv4: true,
			IPv6: true,
		},
		{
			Name: "ipinfo",
			URL:  "https://ipinfo.io/ip",
			IPv4: true,
			IPv6: true,
		},
		{
			Name: "ipify",
			URL:  "https://api.ipify.org",
			IPv4: true,
		},
		{
			Name: "ipify6",
			URL:  "https://api6.ipify.org",
			IPv6: true,
		},
		{
			Name: "ddnss4",
			URL:  "https://ip4.ddnss.de/meineip.php",
			IPv4: true,
		},
		{
			Name: "ddnss6",
			URL:  "https://ip6.ddnss.de/meineip.php",
			IPv6: true,
		},
		{
			Name: "google",
			URL:  "https://domains.google.com/checkip",
			IPv4: true,
			IPv6: true,
		},
	}
}
