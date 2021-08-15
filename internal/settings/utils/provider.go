package utils

import (
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

func ToString(domain, host string, provider models.Provider, ipVersion ipversion.IPVersion) string {
	return "[domain: " + domain + " | host: " + host + " | provider: " +
		string(provider) + " | ip: " + ipVersion.String() + "]"
}
