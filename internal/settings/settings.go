package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"strings"

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

func toString(domain, host string, provider models.Provider, ipVersion ipversion.IPVersion) string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: %s | ip: %s]", domain, host, provider, ipVersion)
}

func bodyToSingleLine(body io.Reader) (s string) {
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return ""
	}
	data := string(b)
	return bodyDataToSingleLine(data)
}

func bodyDataToSingleLine(data string) (line string) {
	data = strings.ReplaceAll(data, "\n", "")
	data = strings.ReplaceAll(data, "\r", "")
	data = strings.ReplaceAll(data, "  ", " ")
	data = strings.ReplaceAll(data, "  ", " ")
	return data
}
