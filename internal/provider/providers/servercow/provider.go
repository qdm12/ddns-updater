package servercow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	username      string
	host          string
	domain        string
	ipVersion     ipversion.IPVersion
	password      string
	useProviderIP bool
	ttl           uint
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (
	p *Provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		Domain        string `json:"domain"`
		TTL           uint   `json:"ttl"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	p = &Provider{
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
		domain:        extraSettings.Domain,
		ttl:           extraSettings.TTL,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.username == "":
		return fmt.Errorf("%w", errors.ErrEmptyUsername)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	}
	if strings.Contains(p.host, "*") {
		return fmt.Errorf("%w", errors.ErrHostWildcard)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString("servercow.de", p.host, constants.Servercow, p.ipVersion)
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
		Provider:  "<a href=\"https://servercow.de\">Servercow</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.servercow.de",
		Path:   "/dns/v1/domains/" + p.domain,
	}

	updateHost := p.host
	if updateHost == "@" {
		updateHost = ""
	}

	requestData := struct {
		Type    string `json:"type"`    // constants.A or constants.AAAA depending on ip address given
		Name    string `json:"name"`    // DNS record name (only the subdomain part)
		Content string `json:"content"` // ip address
		TTL     uint   `json:"ttl"`
	}{
		Type:    recordType,
		Name:    updateHost,
		Content: ip.String(),
		TTL:     p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, err
	}
	headers.SetContentType(request, "application/json")
	headers.SetXAuthUsername(request, p.username)
	headers.SetXAuthPassword(request, p.password)
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode > http.StatusUnsupportedMediaType {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)

	var parsedJSON struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}

	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	if parsedJSON.Message != "ok" {
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, parsedJSON.Error)
	}

	return ip, nil
}
