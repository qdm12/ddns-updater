package gandi

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

type Provider struct {
	domain    string
	host      string
	ttl       int
	ipVersion ipversion.IPVersion
	key       string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Key string `json:"key"`
		TTL int    `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
		ttl:       extraSettings.TTL,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	if p.key == "" {
		return fmt.Errorf("%w", errors.ErrEmptyKey)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Gandi, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
	return p.host
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.gandi.net/\">gandi</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	request.Header.Set("X-Api-Key", p.key)
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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
		Path:   fmt.Sprintf("/api/v5/domains/%s/records/%s/%s", p.domain, p.host, recordType),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	const defaultTTL = 3600
	ttl := defaultTTL
	if p.ttl != 0 {
		ttl = p.ttl
	}
	requestData := struct {
		Values [1]string `json:"rrset_values"`
		TTL    int       `json:"rrset_ttl"`
	}{
		Values: [1]string{ipStr},
		TTL:    ttl,
	}
	err = encoder.Encode(requestData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	p.setHeaders(request)

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
