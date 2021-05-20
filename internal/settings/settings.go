package settings

import (
	"context"
	"encoding/json"
	"net"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Settings interface {
	String() string
	Domain() string
	Host() string
	BuildDomainName() string
	HTML() models.HTMLRow
	Proxied() bool
	IPVersion() ipversion.IPVersion
	Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error)
}

type Constructor func(data json.RawMessage, domain string, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (s Settings, err error)
