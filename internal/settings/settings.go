package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
)

type Settings interface {
	String() string
	Domain() string
	Host() string
	BuildDomainName() string
	HTML() models.HTMLRow
	DNSLookup() bool
	IPVersion() models.IPVersion
	Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error)
}

type Constructor func(data json.RawMessage, domain string, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error)

func buildDomainName(host, domain string) string {
	switch host {
	case "@":
		return domain
	case "*":
		return "any." + domain
	default:
		return host + "." + domain
	}
}

func toString(domain, host string, provider models.Provider, ipVersion models.IPVersion) string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: %s | ip: %s]", domain, host, provider, ipVersion)
}
