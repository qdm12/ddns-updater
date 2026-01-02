package apertodns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	providerErrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	token      string
	baseURL    string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	*Provider, error) {
	extraSettings := struct {
		Token   string `json:"token"`
		BaseURL string `json:"base_url"`
	}{}
	err := json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	baseURL := extraSettings.BaseURL
	if baseURL == "" {
		baseURL = "https://api.apertodns.com"
	}

	p := &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
		baseURL:    baseURL,
	}

	err = p.isValid()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.token == "":
		return fmt.Errorf("%w", providerErrors.ErrTokenNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.ApertoDNS, p.ipVersion)
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
		Provider:  "<a href=\"https://apertodns.com\">ApertoDNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)
}

// Update implements the ApertoDNS Protocol v1.2 with intelligent fallback.
// It first tries the modern JSON API, and falls back to legacy DynDNS2
// only for infrastructure errors (not for auth/validation errors).
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (netip.Addr, error) {
	// 1. Try modern protocol first
	newIP, err := p.updateModern(ctx, client, ip)
	if err == nil {
		return newIP, nil
	}

	// 2. Do NOT fallback for "real" errors that would fail on both endpoints
	if errors.Is(err, providerErrors.ErrAuth) ||
		errors.Is(err, providerErrors.ErrHostnameNotExists) ||
		errors.Is(err, providerErrors.ErrBannedAbuse) ||
		errors.Is(err, providerErrors.ErrBadRequest) {
		return netip.Addr{}, err
	}

	// 3. Fallback to legacy DynDNS2 for infrastructure errors
	// (e.g., 404 endpoint not found, 500 server error, network issues)
	return p.updateLegacy(ctx, client, ip)
}

// updateModern uses the ApertoDNS Protocol v1.2 modern JSON API.
// Endpoint: POST /.well-known/apertodns/v1/update
// Auth: Bearer token
func (p *Provider) updateModern(ctx context.Context, client *http.Client, ip netip.Addr) (netip.Addr, error) {
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parsing base URL: %w", err)
	}
	u.Path = "/.well-known/apertodns/v1/update"

	hostname := utils.BuildDomainName(p.owner, p.domain)

	// Build request body
	requestData := struct {
		Hostname string  `json:"hostname"`
		IPv4     *string `json:"ipv4,omitempty"`
		IPv6     *string `json:"ipv6,omitempty"`
	}{
		Hostname: hostname,
	}

	ipStr := ip.String()
	if ip.Is4() {
		requestData.IPv4 = &ipStr
	} else {
		requestData.IPv6 = &ipStr
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("JSON encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	// Parse JSON response
	decoder := json.NewDecoder(response.Body)
	var apiResponse struct {
		Success bool `json:"success"`
		Data    *struct {
			Hostname string  `json:"hostname"`
			IPv4     *string `json:"ipv4"`
			IPv6     *string `json:"ipv6"`
		} `json:"data"`
		Error *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	err = decoder.Decode(&apiResponse)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json decoding response body: %w", err)
	}

	// Handle error responses
	if !apiResponse.Success {
		if apiResponse.Error == nil {
			return netip.Addr{}, fmt.Errorf("%w: unknown error", providerErrors.ErrUnsuccessful)
		}

		switch apiResponse.Error.Code {
		case "invalid_token", "unauthorized":
			return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrAuth, apiResponse.Error.Message)
		case "hostname_not_found":
			return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrHostnameNotExists, apiResponse.Error.Message)
		case "invalid_hostname", "not_fqdn":
			return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrBadRequest, apiResponse.Error.Message)
		case "rate_limited":
			return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrBannedAbuse, apiResponse.Error.Message)
		case "invalid_ip":
			return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrBadRequest, apiResponse.Error.Message)
		case "server_error":
			return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrUnknownResponse, apiResponse.Error.Message)
		default:
			return netip.Addr{}, fmt.Errorf("%w: %s: %s",
				providerErrors.ErrUnsuccessful, apiResponse.Error.Code, apiResponse.Error.Message)
		}
	}

	// Handle success response
	if apiResponse.Data == nil {
		return netip.Addr{}, fmt.Errorf("%w: missing data in response", providerErrors.ErrUnknownResponse)
	}

	// Get the returned IP based on what we sent
	var returnedIPStr string
	if ip.Is4() {
		if apiResponse.Data.IPv4 == nil {
			return netip.Addr{}, fmt.Errorf("%w: missing ipv4 in response", providerErrors.ErrUnknownResponse)
		}
		returnedIPStr = *apiResponse.Data.IPv4
	} else {
		if apiResponse.Data.IPv6 == nil {
			return netip.Addr{}, fmt.Errorf("%w: missing ipv6 in response", providerErrors.ErrUnknownResponse)
		}
		returnedIPStr = *apiResponse.Data.IPv6
	}

	newIP, err := netip.ParseAddr(returnedIPStr)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrIPReceivedMalformed, returnedIPStr)
	}

	if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent %s but received %s",
			providerErrors.ErrIPReceivedMismatch, ip, newIP)
	}

	return newIP, nil
}

// updateLegacy uses the DynDNS2 compatible endpoint (Layer 1).
// Endpoint: GET /nic/update
// Auth: Basic (username="token", password=token)
func (p *Provider) updateLegacy(ctx context.Context, client *http.Client, ip netip.Addr) (netip.Addr, error) {
	u, err := url.Parse(p.baseURL)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parsing base URL: %w", err)
	}
	u.Path = "/nic/update"

	hostname := utils.BuildDomainName(p.owner, p.domain)
	values := url.Values{}
	values.Set("hostname", hostname)
	values.Set("myip", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}
	headers.SetUserAgent(request)
	request.SetBasicAuth("token", p.token)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	s, err := utils.ReadAndCleanBody(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}

	switch response.StatusCode {
	case http.StatusOK:
		// Continue processing
	case http.StatusUnauthorized:
		return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrAuth, s)
	default:
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			providerErrors.ErrHTTPStatusNotValid, response.StatusCode, s)
	}

	switch {
	case s == "":
		return netip.Addr{}, fmt.Errorf("%w: empty response", providerErrors.ErrUnknownResponse)
	case s == "badauth":
		return netip.Addr{}, fmt.Errorf("%w", providerErrors.ErrAuth)
	case s == "nohost":
		return netip.Addr{}, fmt.Errorf("%w", providerErrors.ErrHostnameNotExists)
	case s == "notfqdn":
		return netip.Addr{}, fmt.Errorf("%w: hostname is not a valid FQDN", providerErrors.ErrBadRequest)
	case s == "abuse":
		return netip.Addr{}, fmt.Errorf("%w", providerErrors.ErrBannedAbuse)
	case s == "911":
		return netip.Addr{}, fmt.Errorf("%w: server error, retry later", providerErrors.ErrUnknownResponse)
	}

	var returnedIP string
	if n, _ := fmt.Sscanf(s, "good %s", &returnedIP); n == 1 {
		// IP was updated
	} else if n, _ := fmt.Sscanf(s, "nochg %s", &returnedIP); n == 1 {
		// IP unchanged
	} else {
		return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrUnknownResponse, s)
	}

	newIP, err := netip.ParseAddr(returnedIP)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %s", providerErrors.ErrIPReceivedMalformed, returnedIP)
	}

	if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent %s but received %s",
			providerErrors.ErrIPReceivedMismatch, ip, newIP)
	}

	return newIP, nil
}
