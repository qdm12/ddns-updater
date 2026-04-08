package hetznercloud

import (
	"context"
	"encoding/json"
	stderrors "errors"
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
	domain         string
	owner          string
	ipVersion      ipversion.IPVersion
	ipv6Suffix     netip.Prefix
	token          string
	zoneIdentifier string // optional: wird per API ermittelt wenn leer
	ttl            uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error,
) {
	extraSettings := struct {
		Token          string `json:"token"`
		ZoneIdentifier string `json:"zone_identifier"`
		TTL            uint32 `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}

	ttl := uint32(3600)
	if extraSettings.TTL > 0 {
		ttl = extraSettings.TTL
	}

	err = validateSettings(domain, extraSettings.Token)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:         domain,
		owner:          owner,
		ipVersion:      ipVersion,
		ipv6Suffix:     ipv6Suffix,
		token:          extraSettings.Token,
		zoneIdentifier: extraSettings.ZoneIdentifier,
		ttl:            ttl,
	}, nil
}

func validateSettings(domain, token string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	if token == "" {
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
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
		Provider:  "<a href=\"https://www.hetzner.com\">Hetzner Cloud</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// Zone-ID ermitteln (aus Config oder per API-Lookup)
	zoneID := p.zoneIdentifier
	if zoneID == "" {
		zoneID, err = p.getZoneID(ctx, client)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("getting zone id: %w", err)
		}
	}

	// Aktuellen RRset prüfen
	currentIP, err := p.getRRSet(ctx, client, zoneID, ip)
	switch {
	case stderrors.Is(err, errors.ErrReceivedNoResult):
		// Kein RRset vorhanden → neu anlegen via upsert
	case err != nil:
		return netip.Addr{}, fmt.Errorf("getting rrset: %w", err)
	case currentIP.Compare(ip) == 0:
		return ip, nil // bereits aktuell
	}

	newIP, err = p.setRRSet(ctx, client, zoneID, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("setting rrset: %w", err)
	}
	return newIP, nil
}
