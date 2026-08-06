package hostinger

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain string
	owner  string

	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix

	token string
	ttl   uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix,
) (p *Provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
		TTL   uint32 `json:"ttl"`
	}{}

	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	const defaultTTL uint32 = 14400
	if extraSettings.TTL == 0 {
		extraSettings.TTL = defaultTTL
	}

	err = validateSettings(domain, extraSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, token string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if token == "" {
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Hostinger, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://www.hostinger.com/\">Hostinger</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	type recordContent struct {
		Content string `json:"content"`
	}

	type zoneRecord struct {
		Name    string          `json:"name"`
		Records []recordContent `json:"records"`
		TTL     uint32          `json:"ttl"`
		Type    string          `json:"type"`
	}

	type updateRequest struct {
		Overwrite bool         `json:"overwrite"`
		Zone      []zoneRecord `json:"zone"`
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	requestData := updateRequest{
		Overwrite: true,
		Zone: []zoneRecord{
			{
				Name: p.owner,
				Records: []recordContent{
					{Content: ip.String()},
				},
				TTL:  p.ttl,
				Type: recordType,
			},
		},
	}

	buffer := bytes.NewBuffer(nil)
	err = json.NewEncoder(buffer).Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
	}

	u := url.URL{
		Scheme: "https",
		Host:   "developers.hostinger.com",
		Path:   "/api/dns/v1/zones/" + p.domain,
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, handleAPIError(response)
	}

	return ip, nil
}

func handleAPIError(response *http.Response) error {
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("reading response body: %w", err)
	}

	apiError := struct {
		Error         string `json:"error"`
		CorrelationID string `json:"correlation_id"`
	}{}
	if err := json.Unmarshal(body, &apiError); err != nil {
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid, response.StatusCode, string(body))
	}

	detail := apiError.Error
	if apiError.CorrelationID != "" {
		detail += " (correlation ID: " + apiError.CorrelationID + ")"
	}

	switch response.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("%w: %s", errors.ErrAuth, detail)
	case http.StatusNotFound:
		return fmt.Errorf("%w: %s", errors.ErrDomainNotFound, detail)
	case http.StatusUnprocessableEntity:
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, detail)
	case http.StatusTooManyRequests:
		return fmt.Errorf("%w: %s", errors.ErrRateLimit, detail)
	default:
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid, response.StatusCode, detail)
	}
}
