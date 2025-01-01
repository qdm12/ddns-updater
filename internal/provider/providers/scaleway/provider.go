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
	domain 		string
	owner  		string
	ipVersion  	ipversion.IPVersion
	ipv6Suffix 	netip.Prefix
	secretKey   string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
        SecretKey string `json:"secret_key"`
    }
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
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
	}, nil
}

func validateSettings(domain, secretKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
    case secretKey == "":
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
		Domain: fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:  p.Owner(),
		Provider:  "<a href=\"https://www.scaleway.com/\">Scaleway</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update updates the DNS record for the domain using Scaleway's API.
// API documentation: https://www.scaleway.com/en/developers/api/domains-and-dns/#path-records-update-records-within-a-dns-zone
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
    // Construct the URL for the API request
    u := url.URL{
        Scheme: "https",
        Host:   "api.scaleway.com",
        Path:   fmt.Sprintf("/domain/v2beta1/dns-zones/%s/records", p.domain),
        RawQuery: fmt.Sprintf("A=%s", ip.String()),
    }

    // Prepare the request body
    requestBody := map[string]interface{}{
        "changes": []map[string]interface{}{
            {
                "set": map[string]interface{}{
                    "id_fields": map[string]interface{}{
                        "name": "",
                        "type": "A",
                    },
                    "records": []map[string]interface{}{
                        {
                            "data": ip.String(),
                            "ttl":  300,
                        },
                    },
                },
            },
        },
    }
    requestBodyBytes, err := json.Marshal(requestBody)
    if err != nil {
        return netip.Addr{}, fmt.Errorf("json marshal: %w", err)
    }

    // Create the HTTP request
    request, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), bytes.NewReader(requestBodyBytes))
    if err != nil {
        return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
    }
    request.Header.Set("Content-Type", "application/json")
    request.Header.Set("Accept", "application/json")
    request.Header.Set("X-Auth-Token", p.secretKey)
    headers.SetUserAgent(request)

    // Send the request
    response, err := client.Do(request)
    if err != nil {
        return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
    }
    defer response.Body.Close()

    // Read and clean the response body
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
