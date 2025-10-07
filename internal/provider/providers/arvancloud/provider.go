package arvancloud

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
	domain     string
	owner      string
	token      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
		Token string `json:"token"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain, owner, providerSpecificSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain: domain,
		owner:  owner,
		token:  providerSpecificSettings.Token,
	}, nil
}

func validateSettings(domain, owner, token string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case owner == "*":
		return fmt.Errorf("%w", errors.ErrOwnerWildcard)
	case owner == "":
		return fmt.Errorf("%w", errors.ErrDomainNotValid)
	case token == "":
		return fmt.Errorf("%w ", errors.ErrKeyNotValid)
	case !strings.HasPrefix(token, "apikey "):
		return fmt.Errorf("%w: token should be like `apikey <your-api-key>`", errors.ErrKeyNotValid)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Arvancloud, p.ipVersion)
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
		Provider:  "<a href=\"https://arvancloud.ir/\">ArvanCloud</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// https://www.arvancloud.ir/api/cdn/4.0#tag/DNS-Management/operation/dns-records.show
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	domainID, err := p.getDomainID(ctx, client)
	if err != nil {
		return netip.Addr{}, err
	}

	u := url.URL{
		Scheme: "https",
		Host:   "napi.arvancloud.ir",
		Path:   fmt.Sprintf("/cdn/4.0/domains/%s/dns-records/%s", p.domain, domainID),
	}

	payload, err := json.Marshal(struct {
		Name  string `json:"name"`
		Type  string `json:"type"`
		Value []struct {
			IP string `json:"ip"`
		} `json:"value"`
	}{
		Name: p.owner,
		Type: "a",
		Value: []struct {
			IP string `json:"ip"`
		}{
			{
				IP: ip.String(),
			},
		},
	})

	if err != nil {
		return netip.Addr{}, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewReader(payload))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthorization(request, p.token)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(s))
	}

	return ip, nil
}

func (p *Provider) getDomainID(ctx context.Context, client *http.Client) (string, error) {
	u := url.URL{
		Scheme: "https",
		Host:   "napi.arvancloud.ir",
		Path:   fmt.Sprintf("/cdn/4.0/domains/%s/dns-records", p.domain),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthorization(request, p.token)

	response, err := client.Do(request)

	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(s))
	}

	var parsedJSON struct {
		Data []struct {
			ID   string `json:"id"`
			Type string `json:"type"`
			Name string `json:"name"`
		} `json:"data"`
	}

	err = json.Unmarshal([]byte(s), &parsedJSON)
	if err != nil {
		return "", fmt.Errorf("%w: cannot parse json", errors.ErrReceivedNoResult)
	}

	for _, subdomain := range parsedJSON.Data {
		if subdomain.Name == p.owner {
			return subdomain.ID, nil
		}
	}
	return "", fmt.Errorf("%w: domain not found", errors.ErrDomainNotFound)
}
