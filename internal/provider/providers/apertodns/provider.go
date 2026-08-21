package apertodns

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
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain      string
	owner       string
	ipVersion   ipversion.IPVersion
	ipv6Suffix  netip.Prefix
	token       string
	apiEndpoint string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	*Provider, error) {
	extraSettings := struct {
		Token       string `json:"token"`
		APIEndpoint string `json:"api_endpoint"`
	}{}
	err := json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	apiEndpoint := extraSettings.APIEndpoint
	if apiEndpoint == "" {
		apiEndpoint = "https://api.apertodns.com"
	}

	p := &Provider{
		domain:      domain,
		owner:       owner,
		ipVersion:   ipVersion,
		ipv6Suffix:  ipv6Suffix,
		token:       extraSettings.Token,
		apiEndpoint: apiEndpoint,
	}

	err = p.isValid()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Provider) isValid() error {
	if p.token == "" {
		return fmt.Errorf("%w", ddnserrors.ErrTokenNotSet)
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

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (netip.Addr, error) {
	u, err := url.Parse(p.apiEndpoint)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("parsing API endpoint: %w", err)
	}
	u.Path = "/.well-known/apertodns/v1/update"

	requestData := struct {
		Hostname string     `json:"hostname"`
		IPv4     netip.Addr `json:"ipv4,omitzero"`
		IPv6     netip.Addr `json:"ipv6,omitzero"`
	}{
		Hostname: utils.BuildDomainName(p.owner, p.domain),
	}
	if ip.Is4() {
		requestData.IPv4 = ip
	} else {
		requestData.IPv6 = ip
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("json encoding request data: %w", err)
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

	if response.StatusCode >= http.StatusInternalServerError {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			ddnserrors.ErrHTTPStatusNotValid, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var apiResponse struct {
		Success bool `json:"success"`
		Data    *struct {
			Hostname string      `json:"hostname"`
			IPv4     *netip.Addr `json:"ipv4,omitempty"`
			IPv6     *netip.Addr `json:"ipv6,omitempty"`
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

	if !apiResponse.Success {
		if apiResponse.Error == nil {
			return netip.Addr{}, fmt.Errorf("%w: unknown error", ddnserrors.ErrUnsuccessful)
		}
		return netip.Addr{}, responseError(apiResponse.Error.Code, apiResponse.Error.Message)
	}

	if apiResponse.Data == nil {
		return netip.Addr{}, fmt.Errorf("%w: missing data in response", ddnserrors.ErrUnknownResponse)
	}

	returnedIP := apiResponse.Data.IPv6
	if ip.Is4() {
		returnedIP = apiResponse.Data.IPv4
	}
	if returnedIP == nil {
		return netip.Addr{}, fmt.Errorf("%w: missing IP address in response", ddnserrors.ErrUnknownResponse)
	}

	newIP := *returnedIP
	if ip != newIP {
		return netip.Addr{}, fmt.Errorf("%w: sent %s but received %s",
			ddnserrors.ErrIPReceivedMismatch, ip, newIP)
	}

	return newIP, nil
}

func responseError(code, message string) error {
	switch code {
	case "invalid_token", "unauthorized":
		return fmt.Errorf("%w: %s", ddnserrors.ErrAuth, message)
	case "hostname_not_found":
		return fmt.Errorf("%w: %s", ddnserrors.ErrHostnameNotExists, message)
	case "invalid_hostname", "not_fqdn", "invalid_ip":
		return fmt.Errorf("%w: %s", ddnserrors.ErrBadRequest, message)
	case "rate_limited":
		return fmt.Errorf("%w: %s", ddnserrors.ErrBannedAbuse, message)
	case "server_error":
		return fmt.Errorf("%w: %s", ddnserrors.ErrUnknownResponse, message)
	default:
		return fmt.Errorf("%w: %s: %s", ddnserrors.ErrUnsuccessful, code, message)
	}
}
