package nextdns

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	endpointID string
	apiGUID    string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	if utils.BuildDomainName(owner, domain) != "link-ip.nextdns.io" {
		return nil, fmt.Errorf("%w: domain must be link-ip.nextdns.io, got %s",
			errors.ErrDomainNotValid, utils.BuildDomainName(owner, domain))
	}

	var providerSpecificSettings struct {
		EndpointID string `json:"endpoint_id"`
		APIGUID    string `json:"api_guid"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(providerSpecificSettings.EndpointID, providerSpecificSettings.APIGUID)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		endpointID: providerSpecificSettings.EndpointID,
		apiGUID:    providerSpecificSettings.APIGUID,
	}, nil
}

func validateSettings(endpointID, apiGUID string) error {
	if endpointID == "" {
		return fmt.Errorf("%w: endpoint_id is not set", errors.ErrEndpointNotValid)
	}
	if apiGUID == "" {
		return fmt.Errorf("%w: api_guid is not set", errors.ErrEndpointNotValid)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString("nextdns.io", "link-ip", constants.NextDNS, p.ipVersion)
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
		Provider:  "<a href=\"https://nextdns.io/\">NextDNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   p.BuildDomainName(),
		Path:   fmt.Sprintf("/%s/%s", p.endpointID, p.apiGUID),
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	switch response.StatusCode {
	case http.StatusOK:
		return ip, nil
	case http.StatusBadRequest:
		return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrBadRequest, utils.ToSingleLine(s))
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(s))
	}
}
