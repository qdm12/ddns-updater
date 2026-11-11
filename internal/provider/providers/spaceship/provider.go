package spaceship

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"sync/atomic"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
	apiSecret  string
	ttl        uint32
	knownTTL   atomic.Uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
		TTL       uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.APIKey, extraSettings.APISecret, extraSettings.TTL)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     extraSettings.APIKey,
		apiSecret:  extraSettings.APISecret,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, apiKey, apiSecret string, ttl uint32) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("domain is not valid: %w", err)
	}

	const minTTL, maxTTL = 60, 3600
	switch {
	case apiKey == "":
		return errors.New("API key is not set")
	case apiSecret == "":
		return errors.New("API secret is not set")
	case ttl != 0 && ttl < minTTL:
		return fmt.Errorf("TTL is too low: ttl must be at least %d seconds", minTTL)
	case ttl > maxTTL:
		return fmt.Errorf("TTL is too high: ttl must be at most %d seconds", maxTTL)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Spaceship, p.ipVersion)
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
		Provider: fmt.Sprintf(
			"<a href=\"https://www.spaceship.com/application/advanced-dns-application/manage/%s\">Spaceship</a>",
			p.domain,
		),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetXAPIKey(request, p.apiKey)
	headers.SetXAPISecret(request, p.apiSecret)
}

func (p *Provider) handleAPIError(response *http.Response) error {
	var apiError apiError
	if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
		return fmt.Errorf("HTTP status is not valid: %d", response.StatusCode)
	}

	errorCode := response.Header.Get("Spaceship-Error-Code")

	switch response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("bad authentication: %s", errorCode)
	case http.StatusForbidden:
		return errors.New("bad authentication: possibly missing dnsRecords:write permission")
	case http.StatusNotFound:
		if apiError.Detail == "SOA record for domain "+p.domain+" not found." {
			return errors.New("domain not found: domain must be configured in Spaceship first")
		}
		return fmt.Errorf("%w: %s", ddnserrors.ErrRecordResourceSetNotFound, apiError.Detail)
	case http.StatusBadRequest:
		var errorDetails string
		if errorCode != "" {
			errorDetails = "(code: " + errorCode + ")"
		}
		fieldDetails := make([]string, 0, len(apiError.Data))
		for _, d := range apiError.Data {
			if d.Field == "" {
				fieldDetails = append(fieldDetails, d.Details)
				continue
			}
			fieldDetails = append(fieldDetails, fmt.Sprintf("%s: %s", d.Field, d.Details))
		}
		if len(fieldDetails) > 0 {
			errorDetails += " - " + strings.Join(fieldDetails, "; ")
		}
		return fmt.Errorf("bad request sent: %s", errorDetails)
	case http.StatusTooManyRequests:
		return errors.New("rate limit exceeded (300 requests/300 seconds) for this domain")
	default:
		var detail string
		if errorCode != "" {
			detail = "(code " + errorCode + "): "
		}
		detail += apiError.Detail
		return fmt.Errorf("HTTP status is not valid: %d: %s", response.StatusCode, detail)
	}
}
