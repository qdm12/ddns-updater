package params

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func settingsIPVersionChecks(ipVersion models.IPVersion, provider models.Provider) error {
	switch ipVersion {
	case constants.IPv4OrIPv6, constants.IPv4:
	case constants.IPv6:
		switch provider {
		case constants.GODADDY, constants.DNSPOD, constants.DREAMHOST, constants.DUCKDNS, constants.NOIP:
			return fmt.Errorf("IPv6 support for %s is not supported yet", provider)
		}
	default:
		return fmt.Errorf("ip version %q is not valid", ipVersion)
	}
	return nil
}
