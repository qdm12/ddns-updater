package publicip

import (
	"net/http"

	"github.com/qdm12/ddns-updater/pkg/publicip/dns"
	iphttp "github.com/qdm12/ddns-updater/pkg/publicip/http"
)

type settings struct {
	// If both dns and http are enabled it will cycle between both of them.
	dns       DNSSettings
	http      HTTPSettings
	privateIP PrivateIPSettings
}

type DNSSettings struct {
	Enabled bool
	Options []dns.Option
}

type HTTPSettings struct {
	Enabled bool
	Client  *http.Client
	Options []iphttp.Option
}

type PrivateIPSettings struct {
	Enabled bool
}
