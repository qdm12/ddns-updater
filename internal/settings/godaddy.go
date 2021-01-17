package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
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

func NewGodaddy(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
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
		return ErrMalformedKey
	case !g.matcher.GodaddySecret(g.secret):
		return ErrMalformedSecret
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

func (g *godaddy) setHeaders(request *http.Request) {
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	request.Header.Set("Authorization", "sso-key "+g.key+":"+g.secret)
}

func (g *godaddy) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	requestData := []goDaddyPutBody{
		{Data: ip.String()},
	}
	if err := encoder.Encode(requestData); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	g.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}

	err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, response.StatusCode)
	var parsedJSON struct {
		Message string `json:"message"`
	}
	decoder := json.NewDecoder(response.Body)
	jsonErr := decoder.Decode(&parsedJSON)
	if jsonErr != nil || len(parsedJSON.Message) == 0 {
		return nil, err
	}
	return nil, fmt.Errorf("%w: %s", err, parsedJSON.Message)
}
