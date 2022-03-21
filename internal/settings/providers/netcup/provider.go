package netcup

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	customerNumber string
	domain         string
	host           string
	ipVersion      ipversion.IPVersion
	apiKey         string
	password       string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		CustomerNumber string `json:"customer_number"`
		ApiKey         string `json:"api_key"`
		Password       string `json:"password"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		customerNumber: extraSettings.CustomerNumber,
		apiKey:         extraSettings.ApiKey,
		password:       extraSettings.Password,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case p.customerNumber == "":
		return errors.ErrEmptyCustomerNumber
	case p.apiKey == "":
		return errors.ErrEmptyAppKey
	case p.password == "":
		return errors.ErrEmptyPassword
	case p.host == "*":
		return errors.ErrHostWildcard
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Netcup, p.ipVersion)
}

func (p *provider) Domain() string {
	return p.domain
}

func (p *provider) Host() string {
	return p.host
}

func (p *provider) IPVersion() ipversion.IPVersion {
	return p.ipVersion
}

func (p *provider) Proxied() bool {
	return false
}

func (p *provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.netcup.eu/\">Netcup.eu</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme:   "https",
		Host:     "ccp.netcup.net",
		Path:     "/run/webservice/servers/endpoint.php",
		RawQuery: "JSON",
	}
	nc := NewClient(p.customerNumber, p.apiKey, p.password, u.String())

	err = nc.Login(ctx)
	if err != nil {
		return netip.Addr{}, err
	}
	fmt.Println("Try to get record to update: ")

	record, err := nc.GetRecordToUpdate(ctx, p.domain, p.host, ip)
	if err != nil {
		return netip.Addr{}, err
	}
	fmt.Println(record)
	fmt.Println("------------------")

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	// if record == nil { // Otherwise the record gets set two times.
	// 	record = NewDNSRecord(p.host, recordType, ip.String())
	// }

	var updateRecords []DNSRecord
	updateRecords = append(updateRecords, *record)
	updateRecordSet := NewDNSRecordSet(updateRecords)
	fmt.Println("UpdateRecordSet: ", updateRecordSet)
	fmt.Println("UpdateRecords: ", updateRecords)
	updated, err := nc.UpdateDNSRecords(ctx, p.domain, updateRecordSet)
	if err != nil {
		return netip.Addr{}, err
	}
	fmt.Println("Updated: ", updated)

	var result DNSRecordSet
	err = json.Unmarshal(updated.ResponseData, &result)
	if err != nil {
		return netip.Addr{}, err
	}
	fmt.Println("Result: ", result)
	var returnedUpdated = result.GetRecord(p.host, recordType)
	var destination = returnedUpdated.Destination
	fmt.Println("destination: ", destination)
	newIP, err = netip.ParseAddr(destination)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	}
	if ip.Compare(newIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return newIP, nil
}
