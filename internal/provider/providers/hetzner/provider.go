package hetzner

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
	host           string
	ipVersion      ipversion.IPVersion
	ipv6Suffix     netip.Prefix
	token          string
	zoneIdentifier string
	ttl            uint
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	extraSettings := struct {
		Token          string `json:"token"`
		ZoneIdentifier string `json:"zone_identifier"`
		TTL            uint   `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		ipv6Suffix:     ipv6Suffix,
		token:          extraSettings.Token,
		zoneIdentifier: extraSettings.ZoneIdentifier,
		ttl:            extraSettings.TTL,
	}
	if p.ttl == 0 {
		p.ttl = 1
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.zoneIdentifier == "":
		return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
	case p.token == "":
		return fmt.Errorf("%w", errors.ErrTokenNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Hetzner, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Host() string {
	return p.host
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
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.hetzner.com\">Hetzner</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordID, upToDate, err := p.getRecordID(ctx, client, ip)
	switch {
	case stderrors.Is(err, errors.ErrReceivedNoResult):
		err = p.createRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("creating record: %w", err)
		}
		return ip, nil
	case err != nil:
		return netip.Addr{}, fmt.Errorf("getting record id: %w", err)
	case upToDate:
		return ip, nil
	}

	ip, err = p.updateRecord(ctx, client, recordID, ip)
	if err != nil {
		return newIP, fmt.Errorf("updating record: %w", err)
	}

	return ip, nil
}
