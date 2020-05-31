package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/network"
	netlib "github.com/qdm12/golibs/network"
)

type godaddy struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	key       string
	secret    string
}

func NewGodaddy(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
	extraSettings := struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	g := &godaddy{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		dnsLookup: !noDNSLookup,
		key:       extraSettings.Key,
		secret:    extraSettings.Secret,
	}
	if err := g.isValid(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *godaddy) isValid() error {
	switch {
	case !constants.MatchGodaddyKey(g.key):
		return fmt.Errorf("invalid key format")
	case !constants.MatchGodaddySecret(g.secret):
		return fmt.Errorf("invalid secret format")
	}
	return nil
}

func (g *godaddy) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Godaddy]", g.domain, g.host)
}

func (g *godaddy) Domain() string {
	return g.domain
}

func (g *godaddy) Host() string {
	return g.host
}

func (g *godaddy) IPVersion() models.IPVersion {
	return g.ipVersion
}

func (g *godaddy) DNSLookup() bool {
	return g.dnsLookup
}

func (g *godaddy) BuildDomainName() string {
	return buildDomainName(g.host, g.domain)
}

func (g *godaddy) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", g.BuildDomainName(), g.BuildDomainName())),
		Host:      models.HTML(g.Host()),
		Provider:  "<a href=\"https://godaddy.com\">GoDaddy</a>",
		IPVersion: models.HTML(g.ipVersion),
	}
}

func (g *godaddy) Update(client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := "A"
	if ip.To4() == nil {
		recordType = "AAAA"
	}
	type goDaddyPutBody struct {
		Data string `json:"data"` // IP address to update to
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.godaddy.com",
		Path:   fmt.Sprintf("/v1/domains/%s/records/%s/%s", g.domain, recordType, g.host),
	}
	r, err := network.BuildHTTPPut(u.String(), []goDaddyPutBody{{ip.String()}})
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	r.Header.Set("Authorization", "sso-key "+g.key+":"+g.secret)
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		var parsedJSON struct {
			Message string `json:"message"`
		}
		if err := json.Unmarshal(content, &parsedJSON); err != nil {
			return nil, err
		} else if len(parsedJSON.Message) > 0 {
			return nil, fmt.Errorf("HTTP status %d - %s", status, parsedJSON.Message)
		}
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	return ip, nil
}
