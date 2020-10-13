package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	netlib "github.com/qdm12/golibs/network"
)

//nolint:maligned
type digitalOcean struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	token     string
}

func NewDigitalOcean(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &digitalOcean{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		dnsLookup: !noDNSLookup,
		token:     extraSettings.Token,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *digitalOcean) isValid() error {
	if len(d.token) == 0 {
		return fmt.Errorf("token cannot be empty")
	}
	return nil
}

func (d *digitalOcean) String() string {
	return toString(d.domain, d.host, constants.DIGITALOCEAN, d.ipVersion)
}

func (d *digitalOcean) Domain() string {
	return d.domain
}

func (d *digitalOcean) Host() string {
	return d.host
}

func (d *digitalOcean) DNSLookup() bool {
	return d.dnsLookup
}

func (d *digitalOcean) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *digitalOcean) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *digitalOcean) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.digitalocean.com/\">DigitalOcean</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func getRecordID(domain, fqdn, recordType, token string, client netlib.Client) (recordID int, err error) {
	values := url.Values{}
	values.Set("name", fqdn)
	values.Set("type", recordType)
	u := url.URL{
		Scheme:   "https",
		Host:     "api.digitalocean.com",
		Path:     fmt.Sprintf("/v2/domains/%s/records", domain),
		RawQuery: values.Encode(),
	}
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentid.mcgaw@gmail.com")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+token)
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return 0, err
	} else if status != http.StatusOK {
		return 0, fmt.Errorf("cannot get record id: HTTP status code %d", status)
	}
	var result struct {
		DomainRecords []struct {
			ID int `json:"id"`
		} `json:"domain_records"`
	}
	err = json.Unmarshal(content, &result)
	switch {
	case err != nil:
		return 0, err
	case len(result.DomainRecords) == 0:
		return 0, fmt.Errorf("no domain records found")
	case result.DomainRecords[0].ID == 0:
		return 0, fmt.Errorf("ID not found in domain record")
	default:
		return result.DomainRecords[0].ID, nil
	}
}

func (d *digitalOcean) Update(client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil { // IPv6
		recordType = AAAA
	}
	recordID, err := getRecordID(d.domain, d.BuildDomainName(), recordType, d.token, client)
	if err != nil {
		return nil, err
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.digitalocean.com",
		Path:   fmt.Sprintf("/v2/domains/%s/records/%d", d.domain, recordID),
	}
	requestData := struct {
		Type string `json:"type"`
		Name string `json:"name"`
		Data string `json:"data"`
	}{
		Type: recordType,
		Name: d.host,
		Data: ip.String(),
	}
	requestBody, err := json.Marshal(requestData)
	if err != nil {
		return nil, err
	}
	r, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentid.mcgaw@gmail.com")
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+d.token)
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	}
	s := string(content)
	if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d: %s", status, strings.ReplaceAll(s, "\n", ""))
	}
	var responseData struct {
		DomainRecord struct {
			Data string `json:"data"`
		} `json:"domain_record"`
	}
	if err := json.Unmarshal(content, &responseData); err != nil {
		return nil, fmt.Errorf("cannot unmarshal response from API update: %w", err)
	}
	newIP = net.ParseIP(responseData.DomainRecord.Data)
	if newIP == nil {
		return nil, fmt.Errorf("IP address received %q is malformed", responseData.DomainRecord.Data)
	} else if !newIP.Equal(ip) {
		return nil, fmt.Errorf("updated IP address %s is not the IP address %s sent for update", newIP, ip)
	}
	return newIP, nil
}
