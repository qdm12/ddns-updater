package hetznercloud

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
	token      string
	// ttl is the Time To Live for the DNS record in seconds.
	// It is optional, and is ONLY used to add a record to the rrset.
	// See https://docs.hetzner.cloud/reference/cloud#tag/zone-rrset-actions/add_zone_rrset_records.body.ttl
	ttl uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Token string `json:"token"`
		TTL   uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	err = validateSettings(domain, extraSettings.Token, extraSettings.TTL)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		token:      extraSettings.Token,
		ttl:        extraSettings.TTL,
	}, nil
}

func validateSettings(domain, token string, ttl uint32) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case token == "":
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	case ttl != 0:
		const minTTL, maxTTL = 60, 2147483647
		switch {
		case ttl < minTTL:
			return fmt.Errorf("%w: %d must be at least %d seconds", errors.ErrTTLTooLow, ttl, minTTL)
		case ttl > maxTTL:
			return fmt.Errorf("%w: %d must be at most %d seconds", errors.ErrTTLTooHigh, ttl, maxTTL)
		}
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.HetznerCloud, p.ipVersion)
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
		Provider:  "<a href=\"https://www.hetzner.com/cloud/\">Hetzner Cloud</a>",
		IPVersion: p.ipVersion.String(),
	}
}

// Update updates the DNS record with the given IP address.
// It first checks if a record exists and if the IP is up to date.
// If the record doesn't exist, it creates a new one.
// If the record exists but has a different IP, it updates the record.
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	exists, upToDate, err := p.getRecord(ctx, client, ip)
	switch {
	case err != nil:
		return netip.Addr{}, fmt.Errorf("getting record id: %w", err)
	case upToDate:
		return ip, nil
	case exists:
		err = p.setRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("updating record: %w", err)
		}
	default:
		err = p.createRRSet(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
	}
	return ip, nil
}
