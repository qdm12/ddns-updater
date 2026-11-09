package dnsexit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

// Response codes returned by the DNSExit dynamic IP update endpoint.
// See https://dnsexit.com/dns/dns-api/#dynamic-ip-update
const (
	codeSuccess     = 0 // Update successful.
	codeNotChanged  = 1 // IP address not changed, which is not an error for us.
	codeAuthError   = 2 // API key is missing or wrong.
	codeMissingData = 3 // Required definitions are missing.
	codeInvalidJSON = 4 // JSON data syntax error.
	codeUnsupported = 5 // Record type not supported.
	codeSystemError = 6 // System error, or hostname not found.
	codePostData    = 7 // The server failed to receive the posted data.
	codeRateLimited = 8 // Updates too often.
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix,
) (p *Provider, err error) {
	extraSettings := struct {
		APIKey string `json:"api_key"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain, owner, extraSettings.APIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     extraSettings.APIKey,
	}, nil
}

func validateSettings(domain, owner, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("domain is not valid: %w", err)
	}

	switch {
	case owner == "*":
		return errors.New(`owner cannot be "*"`)
	case apiKey == "":
		return errors.New("API key is not set")
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.DNSExit, p.ipVersion)
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
		Provider:  "<a href=\"https://www.dnsexit.com/\">DNSExit</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update sends a dynamic IP update to the DNSExit quick update endpoint,
// documented at https://dnsexit.com/dns/dns-api/#dynamic-ip-update
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dnsexit.com",
		Path:   "/dns/ud/",
	}
	values := url.Values{}
	values.Set("apikey", p.apiKey)
	values.Set("host", p.BuildDomainName())
	values.Set("ip", ip.String())

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(),
		strings.NewReader(values.Encode()))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}
	bodyString := utils.ToSingleLine(string(body))

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("HTTP status is not valid: %d: %s",
			response.StatusCode, bodyString)
	}

	var result struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	err = json.Unmarshal(body, &result)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response %q: %w", bodyString, err)
	}

	switch result.Code {
	case codeSuccess, codeNotChanged:
		return ip, nil
	case codeAuthError:
		return netip.Addr{}, fmt.Errorf("bad authentication: %s", result.Message)
	case codeMissingData, codeInvalidJSON:
		return netip.Addr{}, fmt.Errorf("bad request sent: %s", result.Message)
	case codeUnsupported:
		return netip.Addr{}, fmt.Errorf("feature is not available to the user: %s", result.Message)
	case codeSystemError:
		if strings.Contains(strings.ToLower(result.Message), "fail to find") {
			return netip.Addr{}, fmt.Errorf("hostname does not exist: %s", result.Message)
		}
		return netip.Addr{}, fmt.Errorf("server side DNS error: %s", result.Message)
	case codePostData:
		return netip.Addr{}, fmt.Errorf("server side DNS error: %s", result.Message)
	case codeRateLimited:
		return netip.Addr{}, fmt.Errorf("rate limit exceeded: %s", result.Message)
	default:
		return netip.Addr{}, fmt.Errorf("unknown response received: code %d: %s",
			result.Code, result.Message)
	}
}
