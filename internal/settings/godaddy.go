package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/network"
	"github.com/qdm12/ddns-updater/internal/regex"
	netlib "github.com/qdm12/golibs/network"
)

type godaddy struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	key       string
	secret    string
	matcher   regex.Matcher
}

func NewGodaddy(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
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
		matcher:   matcher,
	}
	if err := g.isValid(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *godaddy) isValid() error {
	switch {
	case !g.matcher.GodaddyKey(g.key):
		return fmt.Errorf("invalid key format")
	case !g.matcher.GodaddySecret(g.secret):
		return fmt.Errorf("invalid secret format")
	}
	return nil
}

func (g *godaddy) String() string {
	return toString(g.domain, g.host, constants.GODADDY, g.ipVersion)
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

func (g *godaddy) Update(ctx context.Context, client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
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
	r = r.WithContext(ctx)
	content, status, err := client.Do(r)
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
		return nil, fmt.Errorf(http.StatusText(status))
	}
	return ip, nil
}
