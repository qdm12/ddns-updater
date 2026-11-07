package enum

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"sync/atomic"

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
	projectID  string
	ttl        uint32
	knownTTL   atomic.Uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		APIKey    string `json:"api_key"`
		ProjectID string `json:"project_id"`
		TTL       uint32 `json:"ttl"`
	}{}

	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.APIKey, extraSettings.ProjectID, extraSettings.TTL)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     extraSettings.APIKey,
		projectID:  extraSettings.ProjectID,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, apiKey, projectID string, ttl uint32) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	const minTTL, maxTTL = 60, 86400
	switch {
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case projectID == "":
		return fmt.Errorf("%w", errors.ErrProjectIDNotSet)
	case ttl != 0 && ttl < minTTL:
		return fmt.Errorf("%w: %d must be at least %d seconds", errors.ErrTTLTooLow, ttl, minTTL)
	case ttl > maxTTL:
		return fmt.Errorf("%w: %d must be at most %d seconds", errors.ErrTTLTooHigh, ttl, maxTTL)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Enum, p.ipVersion)
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
		Provider:  "<a href=\"https://enum.co/\">enum</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// TTL returns the record TTL in seconds if it is known.
func (p *Provider) TTL() (ttl *uint32) {
	if p.ttl != 0 {
		return &p.ttl
	}

	if knownTTL := p.knownTTL.Load(); knownTTL != 0 {
		return &knownTTL
	}

	return nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	zoneID, err := p.getZoneID(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting zone id: %w", err)
	}

	exists, upToDate, currentTTL, err := p.getRecordSet(ctx, client, zoneID, recordType, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record set: %w", err)
	}

	ttl := p.ttl
	if ttl == 0 {
		ttl = currentTTL
	}

	switch {
	case upToDate:
		return ip, nil
	case exists:
		err = p.setRecordSet(ctx, client, "UpdateRecordSet", zoneID, recordType, ttl, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("updating record set: %w", err)
		}
	default:
		err = p.setRecordSet(ctx, client, "CreateRecordSet", zoneID, recordType, ttl, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record set: %w", err)
		}
	}
	return ip, nil
}
