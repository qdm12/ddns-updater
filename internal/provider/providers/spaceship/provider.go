package spaceship

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

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
	apiKey     string
	apiSecret  string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		APIKey    string `json:"api_key"`
		APISecret string `json:"api_secret"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.APIKey, extraSettings.APISecret)
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
	}, nil
}

func validateSettings(domain, apiKey, apiSecret string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case apiSecret == "":
		return fmt.Errorf("%w", errors.ErrAPISecretNotSet)
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
	request.Header.Set("X-Api-Key", p.apiKey)
	request.Header.Set("X-Api-Secret", p.apiSecret)
}

func (p *Provider) handleAPIError(response *http.Response) error {
	var apiError APIError
	if err := json.NewDecoder(response.Body).Decode(&apiError); err != nil {
		return fmt.Errorf("%w: %d", errors.ErrHTTPStatusNotValid, response.StatusCode)
	}

	// Extract error code from header if present
	errorCode := response.Header.Get("Spaceship-Error-Code")

	switch response.StatusCode {
	case http.StatusUnauthorized:
		return fmt.Errorf("%w: invalid API credentials", errors.ErrAuth)
	case http.StatusForbidden:
		return fmt.Errorf("%w: missing required permission dnsRecords:write", errors.ErrAuth)
	case http.StatusNotFound:
		if apiError.Detail == "SOA record for domain "+p.domain+" not found." {
			return fmt.Errorf("%w: domain %s must be configured in Spaceship first",
				errors.ErrDomainNotFound, p.domain)
		}
		return fmt.Errorf("%w: %s", errors.ErrRecordResourceSetNotFound, apiError.Detail)
	case http.StatusBadRequest:
		var details string
		for _, d := range apiError.Data {
			if d.Field != "" {
				details += fmt.Sprintf(" %s: %s;", d.Field, d.Details)
			} else {
				details += fmt.Sprintf(" %s;", d.Details)
			}
		}
		// Add error code if present
		if errorCode != "" {
			details = fmt.Sprintf(" (code: %s)%s", errorCode, details)
		}
		return fmt.Errorf("%w:%s", errors.ErrBadRequest, details)
	case http.StatusTooManyRequests:
		// Rate limit is 300 requests within 300 seconds per user and domain
		return fmt.Errorf("%w: rate limit exceeded (300 requests/300 seconds)", errors.ErrRateLimit)
	case http.StatusInternalServerError:
		if errorCode != "" {
			return fmt.Errorf("%w: internal server error (code: %s): %s",
				errors.ErrHTTPStatusNotValid, errorCode, apiError.Detail)
		}
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, apiError.Detail)
	default:
		if errorCode != "" {
			return fmt.Errorf("%w: %d (code: %s): %s",
				errors.ErrHTTPStatusNotValid, response.StatusCode, errorCode, apiError.Detail)
		}
		return fmt.Errorf("%w: %d: %s",
			errors.ErrHTTPStatusNotValid, response.StatusCode, apiError.Detail)
	}
}
