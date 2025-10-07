package porkbun

import (
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

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
	p *Provider, err error,
) {
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
	records, err := p.getRecords(ctx, client, recordType, p.owner)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record IDs: %w", err)
	}

	if len(records) == 0 {
		err = p.deleteDefaultConflictingRecordsIfNeeded(ctx, client)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("deleting default conflicting records: %w", err)
		}

		err = p.createRecord(ctx, client, recordType, p.owner, ipStr)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	for _, record := range records {
		err = p.updateRecord(ctx, client, recordType, p.owner, ipStr, record.ID)
		time.Sleep(time.Second)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("updating record: %w", err)
		}
	}
	return ip, nil
}

// deleteDefaultConflictingRecordsIfNeeded deletes any default records that would conflict with a new record,
// see https://github.com/qdm12/ddns-updater/blob/master/docs/porkbun.md#record-creation
func (p *Provider) deleteDefaultConflictingRecordsIfNeeded(ctx context.Context, client *http.Client) (err error) {
	const porkbunParkedDomain = "pixie.porkbun.com"
	switch p.owner {
	case "@":
		err = p.deleteSingleMatchingRecord(ctx, client, constants.ALIAS, "@", porkbunParkedDomain)
		if err != nil {
			return fmt.Errorf("deleting default ALIAS @ parked domain record: %w", err)
		}
		return nil
	case "*":
		err = p.deleteSingleMatchingRecord(ctx, client, constants.CNAME, "*", porkbunParkedDomain)
		if err != nil {
			return fmt.Errorf("deleting default CNAME * parked domain record: %w", err)
		}

		err = p.deleteSingleMatchingRecord(ctx, client, constants.ALIAS, "@", porkbunParkedDomain)
		if err == nil || stderrors.Is(err, errors.ErrConflictingRecord) {
			// allow conflict ALIAS records to be set to something besides the parked domain
			return nil
		}
		return fmt.Errorf("deleting default ALIAS @ parked domain record: %w", err)
	default:
		return nil
	}
}

// deleteSingleMatchingRecord deletes an eventually present record matching a specific record type if the content
// matches the expected content value.
// It returns an error if multiple records are found or if one record is found with an unexpected value.
func (p *Provider) deleteSingleMatchingRecord(ctx context.Context, client *http.Client,
	recordType, owner, expectedContent string,
) (err error) {
	records, err := p.getRecords(ctx, client, recordType, owner)
	if err != nil {
		return fmt.Errorf("getting records: %w", err)
	}

	switch {
	case len(records) == 0:
		return nil
	case len(records) > 1:
		return fmt.Errorf("%w: %d %s records are already set", errors.ErrConflictingRecord, len(records), recordType)
	case records[0].Content != expectedContent:
		return fmt.Errorf("%w: %s record has content %q mismatching expected content %q",
			errors.ErrConflictingRecord, recordType, records[0].Content, expectedContent)
	}

	// Single record with content matching expected content.
	err = p.deleteRecord(ctx, client, recordType, owner)
	if err != nil {
		return fmt.Errorf("deleting record: %w", err)
	}
	return nil
}
