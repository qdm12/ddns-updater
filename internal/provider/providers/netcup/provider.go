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
	customerNumber string
	domain         string
	host           string
	ipVersion      ipversion.IPVersion
	apiKey         string
	password       string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	var extraSettings struct {
		CustomerNumber string `json:"customer_number"`
		APIKey         string `json:"api_key"`
		Password       string `json:"password"`
	}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, fmt.Errorf("JSON decoding provider specific settings: %w", err)
	}

	p = &Provider{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		customerNumber: extraSettings.CustomerNumber,
		apiKey:         extraSettings.APIKey,
		password:       extraSettings.Password,
	}

	err = p.isValid()
	if err != nil {
		return nil, fmt.Errorf("validating provider settings: %w", err)
	}

	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.customerNumber == "":
		return fmt.Errorf("%w", errors.ErrEmptyCustomerNumber)
	case p.apiKey == "":
		return fmt.Errorf("%w", errors.ErrEmptyAPIKey)
	case p.password == "":
		return fmt.Errorf("%w", errors.ErrEmptyPassword)
	case p.host == "*":
		return fmt.Errorf("%w", errors.ErrHostWildcard)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Netcup, p.ipVersion)
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

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.netcup.eu/\">Netcup.eu</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
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
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUpdateRecord, err)
	}

	found := false
	for _, record = range updateResponse.DNSRecords {
		if record.Hostname == p.host && record.Type == recordType {
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
