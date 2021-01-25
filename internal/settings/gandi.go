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

const DefaultTTL = 3600

type gandi struct {
	domain    string
	host      string
	ttl       int
	ipVersion models.IPVersion
	dnsLookup bool
	key       string
}

func NewGandi(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Key string `json:"key"`
		TTL int    `json:"ttl"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &gandi{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		dnsLookup: !noDNSLookup,
		key:       extraSettings.Key,
		ttl:       extraSettings.TTL,
	}
	if err := g.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (g *gandi) isValid() error {
	if len(g.key) == 0 {
		return ErrEmptyKey
	}
	return nil
}

func (g *gandi) String() string {
	return toString(g.domain, g.host, constants.GANDI, g.ipVersion)
}

func (g *gandi) Domain() string {
	return g.domain
}

func (g *gandi) Host() string {
	return g.host
}

func (g *gandi) DNSLookup() bool {
	return g.dnsLookup
}

func (g *gandi) IPVersion() models.IPVersion {
	return g.ipVersion
}

func (g *gandi) BuildDomainName() string {
	return buildDomainName(g.host, g.domain)
}

func (g *gandi) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", g.BuildDomainName(), g.BuildDomainName())),
		Host:      models.HTML(g.Host()),
		Provider:  "<a href=\"https://www.gandi.net/\">gandi</a>",
		IPVersion: models.HTML(g.ipVersion),
	}
}

func (g *gandi) setHeaders(request *http.Request) {
	setUserAgent(request)
	setContentType(request, "application/json")
	setAccept(request, "application/json")
	request.Header.Set("X-Api-Key", g.key)
}

func (g *gandi) getRecordIP(ctx context.Context, recordType string, client *http.Client) (
	recordIP string, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "dns.api.gandi.net",
		Path:   fmt.Sprintf("/api/v5/domains/%s/records/%s/%s", g.domain, g.host, recordType),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	g.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var result struct {
		Type   string   `json:"rrset_type"`
		TTL    int      `json:"rrset_ttl"`
		Name   string   `json:"rrset_name"`
		Href   string   `json:"rrset_href"`
		Values []string `json:"rrset_values"`
	}
	if err = decoder.Decode(&result); err != nil {
		return "", fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	if len(result.Values) == 0 {
		return "", ErrNoResultReceived
	}
	return result.Values[0], nil
}

func (g *gandi) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil { // IPv6
		recordType = AAAA
	}

	recordIP, err := g.getRecordIP(ctx, recordType, client)
	if err != nil && err != ErrNoResultReceived { // if no ip was defined before, let's proceed with the update
		return nil, fmt.Errorf("%s: %w", ErrGetRecordIP, err)
	}

	oldIP := net.ParseIP(recordIP)
	if ip.Equal(oldIP) { // success, nothing to change
		return ip, nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.api.gandi.net",
		Path:   fmt.Sprintf("/api/v5/domains/%s/records/%s/%s", g.domain, g.host, recordType),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	requestData := struct {
		Values [1]string `json:"rrset_values"`
		TTL    int       `json:"rrset_ttl"`
	}{
		Values: [1]string{ip.To4().String()},
		TTL: func() int {
			ttl := DefaultTTL
			if g.ttl != 0 {
				ttl = g.ttl
			}
			return ttl
		}(),
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

	if response.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	return ip, nil
}