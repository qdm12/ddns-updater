package scaleway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	secretKey  string
	ttl        uint16
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
		SecretKey string `json:"secret_key"`
		TTL       uint16 `json:"ttl"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	if providerSpecificSettings.TTL == 0 {
		providerSpecificSettings.TTL = 3600
	}

	err = validateSettings(domain,
		providerSpecificSettings.SecretKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		secretKey:  providerSpecificSettings.SecretKey,
		ttl:        providerSpecificSettings.TTL,
	}, nil
}

func validateSettings(domain, secretKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if secretKey == "" {
		return fmt.Errorf("%w", errors.ErrSecretKeyNotSet)
	}

	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Dyn, p.ipVersion)
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
		Provider:  "<a href=\"https://www.scaleway.com/\">Scaleway</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update updates the DNS record for the domain using Scaleway's API.
// See https://www.scaleway.com/en/developers/api/domains-and-dns/#path-records-update-records-within-a-dns-zone
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.scaleway.com",
		Path:   fmt.Sprintf("/domain/v2beta1/dns-zones/%s/records", p.domain),
	}

	fieldType := "A"
	if ip.Is6() {
		fieldType = "AAAA"
	}
	type recordJSON struct {
		Data string `json:"data"`
		Name string `json:"name"`
		TTL  uint16 `json:"ttl"`
	}
	type changeJSON struct {
		Set struct {
			IDFields struct {
				Name string `json:"name"`
				Type string `json:"type"`
			} `json:"id_fields"`
			Records []recordJSON `json:"records"`
		} `json:"set"`
	}
	var change changeJSON
	change.Set.IDFields.Name = p.owner
	change.Set.IDFields.Type = fieldType
	change.Set.Records = []recordJSON{{
		Data: ip.String(),
		Name: p.owner,
		TTL:  p.ttl,
	}}
	requestBody := struct {
		Changes []changeJSON `json:"changes"`
	}{
		Changes: []changeJSON{change},
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestBody)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetXAuthToken(request, p.secretKey)
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		var errorResponse struct {
			Message string `json:"message"`
			Type    string `json:"type"`
		}
		if jsonErr := json.Unmarshal([]byte(s), &errorResponse); jsonErr == nil {
			if errorResponse.Type == "denied_authentication" {
				return netip.Addr{}, fmt.Errorf("%w: %s", errors.ErrAuth, errorResponse.Message)
			}
		}
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, utils.ToSingleLine(s))
	}

	return ip, nil
}
