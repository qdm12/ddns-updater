package netcup

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
	domain         string
	owner          string
	ipVersion      ipversion.IPVersion
	ipv6Suffix     netip.Prefix
	customerNumber string
	apiKey         string
	password       string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	var extraSettings struct {
		CustomerNumber string `json:"customer_number"`
		APIKey         string `json:"api_key"`
		Password       string `json:"password"`
	}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding provider specific settings: %w", err)
	}

	err = validateSettings(domain, owner, extraSettings.CustomerNumber,
		extraSettings.APIKey, extraSettings.Password)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	return &Provider{
		domain:         domain,
		owner:          owner,
		ipVersion:      ipVersion,
		ipv6Suffix:     ipv6Suffix,
		customerNumber: extraSettings.CustomerNumber,
		apiKey:         extraSettings.APIKey,
		password:       extraSettings.Password,
	}, nil
}

func validateSettings(domain, owner, customerNumber, apiKey, password string) (err error) {
	err = utils.CheckDomain(domain)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrDomainNotValid, err)
	}

	switch {
	case owner == "*":
		return fmt.Errorf("%w", errors.ErrOwnerWildcard)
	case customerNumber == "":
		return fmt.Errorf("%w", errors.ErrCustomerNumberNotSet)
	case apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case password == "":
		return fmt.Errorf("%w", errors.ErrPasswordNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Netcup, p.ipVersion)
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
		Provider:  "<a href=\"https://www.netcup.eu/\">Netcup.eu</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	session, err := p.login(ctx, client)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("logging in: %w", err)
	}

	record, err := p.getRecordToUpdate(ctx, client, session, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("getting record to update: %w", err)
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	updateRecordSet := dnsRecordSet{
		DNSRecords: []dnsRecord{record},
	}
	updateResponse, err := p.updateDNSRecords(ctx, client, session, updateRecordSet)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record: %w", err)
	}

	found := false
	for _, record = range updateResponse.DNSRecords {
		if record.Hostname == p.owner && record.Type == recordType {
			found = true
			break
		}
	}

	if !found {
		return netip.Addr{}, fmt.Errorf("%w: in %d records from update response data",
			errors.ErrRecordNotFound, len(updateResponse.DNSRecords))
	}

	newIP, err = netip.ParseAddr(record.Destination)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %s",
			errors.ErrIPReceivedMalformed, record.Destination)
	}

	if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: expected %s but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}

	return newIP, nil
}
