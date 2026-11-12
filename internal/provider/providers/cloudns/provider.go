package cloudns

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

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	url        string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	var extraSettings struct {
		URL string `json:"url"`
	}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, errors.New("json decoding provider specific settings: invalid configuration")
	}

	err = utils.CheckDomain(domain)
	if err != nil {
		return nil, fmt.Errorf("domain is not valid: %w", err)
	}
	err = validateURL(extraSettings.URL, ipVersion)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		url:        extraSettings.URL,
	}, nil
}

func validateURL(dynamicURL string, ipVersion ipversion.IPVersion) error {
	if dynamicURL == "" {
		return errors.New("url is not set")
	}
	parsedURL, err := url.Parse(dynamicURL)
	if err != nil {
		// Parse errors can contain the credential-bearing URL.
		return errors.New("url is malformed")
	}
	switch {
	case parsedURL.Scheme != "https":
		return errors.New("url must use HTTPS")
	case parsedURL.User != nil || parsedURL.Fragment != "" || parsedURL.Opaque != "":
		return errors.New("url must not contain user information or a fragment")
	case parsedURL.EscapedPath() != "/api/dynamicURL/":
		return errors.New("url must use the /api/dynamicURL/ endpoint")
	}

	var expectedVersion ipversion.IPVersion
	switch parsedURL.Host {
	case "ipv4.cloudns.net":
		expectedVersion = ipversion.IP4
	case "ipv6.cloudns.net":
		expectedVersion = ipversion.IP6
	default:
		return errors.New("url hostname must be ipv4.cloudns.net or ipv6.cloudns.net")
	}
	if ipVersion != expectedVersion {
		return errors.New("ip_version must explicitly match the url endpoint (ipv4 or ipv6)")
	}

	query, err := url.ParseQuery(parsedURL.RawQuery)
	if err != nil {
		return errors.New("url query is malformed")
	}
	if len(query["q"]) != 1 || strings.TrimSpace(query.Get("q")) == "" {
		return errors.New("url must contain a single non-empty q token")
	}
	if len(query) != 1 {
		return errors.New("url must contain only the q token, without additional parameters")
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.ClouDNS, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://www.cloudns.net/\">ClouDNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update invokes the record-specific Dynamic URL. ClouDNS detects the source IP
// independently; ip is returned for compatibility with the updater interface.
// See https://www.cloudns.net/wiki/article/364/.
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, p.url, nil)
	if err != nil {
		return netip.Addr{}, errors.New("creating http request: invalid Dynamic URL")
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		// HTTP errors can contain the URL, including its secret token.
		switch {
		case errors.Is(err, context.Canceled):
			return netip.Addr{}, fmt.Errorf("doing http request: %w", context.Canceled)
		case errors.Is(err, context.DeadlineExceeded):
			return netip.Addr{}, fmt.Errorf("doing http request: %w", context.DeadlineExceeded)
		default:
			return netip.Addr{}, errors.New("doing http request: network or HTTP client error")
		}
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, errors.New("reading response body")
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return netip.Addr{}, fmt.Errorf("dynamic URL authorization rejected: HTTP status %d", response.StatusCode)
	default:
		return netip.Addr{}, fmt.Errorf("HTTP status is not valid: %d", response.StatusCode)
	}

	// Do not echo response bodies: they could include credentials.
	switch strings.TrimSpace(string(body)) {
	case "OK":
		return ip, nil
	case "Invalid request.":
		return netip.Addr{}, errors.New("invalid Dynamic URL request: check the active record URL and token")
	default:
		return netip.Addr{}, errors.New("unexpected Dynamic URL response: expected OK")
	}
}
