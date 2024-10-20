package vultr

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
	domain     string
	owner      string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	apiKey     string
	ttl        uint32
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error,
) {
	var providerSpecificSettings struct {
		APIKey string `json:"apikey"`
		TTL    uint32 `json:"ttl"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain, providerSpecificSettings.APIKey)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:     domain,
		owner:      owner,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		apiKey:     providerSpecificSettings.APIKey,
		ttl:        providerSpecificSettings.TTL,
	}, nil
}

func validateSettings(domain, apiKey string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Vultr, p.ipVersion)
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
		Provider:  fmt.Sprintf("<a href=\"https://my.vultr.com/dns/edit/?domain=%s#dns-records\">Vultr</a>", p.domain),
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.apiKey)
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	r, err := p.getRecord(ctx, client, recordType)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("error getting records for %s: %w", p.domain, err)
	}

	/*
		1. if there's no record, create it
		2. if it exists and ip is different, update it
		3. if it exists and ip is the same, do nothing
	*/

	if r == (Record{}) {
		err := p.createRecord(ctx, client, ip)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("error creating record: %w", err)
		}
		return ip, nil
	}

	if r.Data != ip.String() {
		newIp, err := p.updateRecord(ctx, client, ip, r)
		if err != nil {
			return netip.Addr{}, fmt.Errorf("error updating record %s: %w", p.BuildDomainName(), err)
		}
		return newIp, nil
	}

	return ip, nil
}
