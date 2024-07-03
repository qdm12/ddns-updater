package ionos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		APIKey string `json:"api_key"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, fmt.Errorf("decoding ionos extra settings: %w", err)
	}

	err = validateSettings(domain, extraSettings.APIKey)
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

func validateSettings(domain, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if apiKey == "" {
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Ionos, p.ipVersion)
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
		Provider:  "<a href=\"https://www.ionos.com/\">Ionos</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// See https://developer.hosting.ionos.com/docs/dns
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (
	newIP netip.Addr, err error) {
	zones, err := p.getZones(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting zones: %w", err)
	}

	var zoneID string
	for _, zone := range zones {
		if zone.Name == p.domain {
			zoneID = zone.ID
			break
		}
	}

	if zoneID == "" {
		return netip.Addr{}, fmt.Errorf("%w: in %d zones for domain %s",
			errors.ErrZoneNotFound, len(zones), p.domain)
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	records, err := p.getRecords(ctx, client, zoneID, recordType)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting records: %w", err)
	}

	const usualRecordsCount = 1
	matchingRecords := make([]apiRecord, 0, usualRecordsCount)
	fullDomainName := utils.BuildURLQueryHostname(p.owner, p.domain)
	for _, record := range records {
		if record.Name == fullDomainName {
			matchingRecords = append(matchingRecords, record)
		}
	}

	if len(matchingRecords) == 0 {
		err = p.createRecord(ctx, client, zoneID, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	}

	for _, matchingRecord := range matchingRecords {
		if matchingRecord.Content == ip.String() {
			continue // already up to date
		}

		err = p.updateRecord(ctx, client, zoneID, matchingRecord, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("updating record: %w", err)
		}
	}

	return ip, nil
}
