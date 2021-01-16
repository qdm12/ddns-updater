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
			Name: "google",
			URL:  "https://domains.google.com/checkip",
			IPv4: true,
			IPv6: true,
		},
		{
			Name: "noip4",
			URL:  "http://ip1.dynupdate.no-ip.com",
			IPv4: true,
		},
		{
			Name: "noip6",
			URL:  "http://ip1.dynupdate6.no-ip.com",
			IPv6: true,
		},
		{
			Name: "noip8245_4",
			URL:  "http://ip1.dynupdate.no-ip.com:8245",
			IPv4: true,
		},
		{
			Name: "noip8245_6",
			URL:  "http://ip1.dynupdate6.no-ip.com:8245",
			IPv6: true,
		},
	}
}
