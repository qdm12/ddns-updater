package porkbun

import (
	"context"
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
	domain       string
	owner        string
	ipVersion    ipversion.IPVersion
	ipv6Suffix   netip.Prefix
	ttl          uint32
	apiKey       string
	secretAPIKey string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		SecretAPIKey string `json:"secret_api_key"`
		APIKey       string `json:"api_key"`
		TTL          uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.APIKey, extraSettings.SecretAPIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:       domain,
		owner:        owner,
		ipVersion:    ipVersion,
		ipv6Suffix:   ipv6Suffix,
		secretAPIKey: extraSettings.SecretAPIKey,
		apiKey:       extraSettings.APIKey,
		ttl:          extraSettings.TTL,
	}, nil
}

func validateSettings(domain, apiKey, secretAPIKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case secretAPIKey == "":
		return fmt.Errorf("%w", errors.ErrAPISecretNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Porkbun, p.ipVersion)
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
		Provider:  "<a href=\"https://www.porkbun.com/\">Porkbun DNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

// See https://porkbun.com/api/json/v3/documentation
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	ipStr := ip.String()
	records, err := p.getRecords(ctx, client, recordType)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record IDs: %w", err)
	}

	if len(records) == 0 {
		// For new domains, Porkbun Creates 2 Default DNS Records which point to their "parked" domain page:
		// ALIAS domain.tld -> pixie.porkbun.com
		// CNAME *.domain.tld -> pixie.porkbun.com
		// ALIAS and CNAME records conflict with A and AAAA records, and attempting to create an A or AAAA record
		// will return a 400 error if they aren't first deleted.
		porkbunParkedDomain := "pixie.porkbun.com"
		switch {
		case p.owner == "@":
			// Delete ALIAS domain.tld -> pixie.porkbun.com record
			err = p.deleteMatchingRecord(ctx, client, constants.ALIAS, porkbunParkedDomain)
			if err != nil {
				return netip.Addr{}, err
			}
		case p.owner == "*":
			// Delete CNAME *.domain.tld -> pixie.porkbun.com record
			err = p.deleteMatchingRecord(ctx, client, constants.CNAME, porkbunParkedDomain)
			if err != nil {
				return netip.Addr{}, err
			}
		}

		err = p.createRecord(ctx, client, recordType, ipStr)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	for _, record := range records {
		err = p.updateRecord(ctx, client, recordType, ipStr, record.ID)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("updating record: %w", err)
		}
	}

	return ip, nil
}

// Deletes the matching record for a specific recordType iff the content matches the expected content value.
// Returns an error if there is a matching record but an unexpected content value.
// If there is no matching record, there is nothing to do.
func (p *Provider) deleteMatchingRecord(
	ctx context.Context,
	client *http.Client,
	recordType string,
	expectedContent string,
) error {
	records, err := p.getRecords(ctx, client, recordType)
	if err != nil {
		return fmt.Errorf("getting %s records: %w", recordType, err)
	}

	if len(records) == 1 && records[0].Content == expectedContent {
		err = p.deleteRecord(ctx, client, recordType)
		if err != nil {
			return fmt.Errorf("deleting %s record: %w", recordType, err)
		}
		return nil
	} else if len(records) >= 1 {
		// We have 1 or more matching records, but the expectedContent didn't match. Throw a conflicting record error.
		return fmt.Errorf("deleting %s record: %w", recordType, errors.ErrConflictingRecord)
	}

	// No record found for recordType, nothing to do.
	return nil
}
