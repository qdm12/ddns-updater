package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type gandi struct {
	domain    string
	host      string
	ttl       int
	ipVersion ipversion.IPVersion
	key       string
}

func NewGandi(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (s Settings, err error) {
	extraSettings := struct {
		Key string `json:"key"`
		TTL int    `json:"ttl"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	g := &gandi{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
		ttl:       extraSettings.TTL,
	}
	if err := g.isValid(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *gandi) isValid() error {
	if len(g.key) == 0 {
		return errors.ErrEmptyKey
	}
	return nil
}

func (g *gandi) String() string {
	return utils.ToString(g.domain, g.host, constants.Gandi, g.ipVersion)
}

func (g *gandi) Domain() string {
	return g.domain
}

func (g *gandi) Host() string {
	return g.host
}

func (g *gandi) IPVersion() ipversion.IPVersion {
	return g.ipVersion
}

func (g *gandi) Proxied() bool {
	return false
}

func (g *gandi) BuildDomainName() string {
	return utils.BuildDomainName(g.host, g.domain)
}

func (g *gandi) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", g.BuildDomainName(), g.BuildDomainName())),
		Host:      models.HTML(g.Host()),
		Provider:  "<a href=\"https://www.gandi.net/\">gandi</a>",
		IPVersion: models.HTML(g.ipVersion.String()),
	}
}

func (g *gandi) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	request.Header.Set("X-Api-Key", g.key)
}

func (g *gandi) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	var ipStr string
	if ip.To4() == nil { // IPv6
		recordType = constants.AAAA
		ipStr = ip.To16().String()
	} else {
		ipStr = ip.To4().String()
	}

	u := url.URL{
		Scheme: "https",
		Host:   "dns.api.gandi.net",
		Path:   fmt.Sprintf("/api/v5/domains/%s/records/%s/%s", g.domain, g.host, recordType),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	const defaultTTL = 3600
	ttl := defaultTTL
	if g.ttl != 0 {
		ttl = g.ttl
	}
	requestData := struct {
		Values [1]string `json:"rrset_values"`
		TTL    int       `json:"rrset_ttl"`
	}{
		Values: [1]string{ipStr},
		TTL:    ttl,
	}
	if err := encoder.Encode(requestData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrRequestEncode, err)
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
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	return ip, nil
}
