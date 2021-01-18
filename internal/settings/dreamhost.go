package settings

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
)

type dreamhost struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	key       string
	matcher   regex.Matcher
}

func NewDreamhost(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Key string `json:"key"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	if len(host) == 0 {
		host = "@" // default
	}
	d := &dreamhost{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		dnsLookup: !noDNSLookup,
		key:       extraSettings.Key,
		matcher:   matcher,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dreamhost) isValid() error {
	switch {
	case !d.matcher.DreamhostKey(d.key):
		return fmt.Errorf("invalid key format")
	case d.host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func (d *dreamhost) String() string {
	return toString(d.domain, d.host, constants.DREAMHOST, d.ipVersion)
}

func (d *dreamhost) Domain() string {
	return d.domain
}

func (d *dreamhost) Host() string {
	return d.host
}

func (d *dreamhost) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *dreamhost) DNSLookup() bool {
	return d.dnsLookup
}

func (d *dreamhost) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *dreamhost) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.dreamhost.com/\">Dreamhost</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *dreamhost) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}

	records, err := d.getRecords(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrListRecords, err)
	}

	var oldIP net.IP
	for _, data := range records.Data {
		if data.Type == recordType && data.Record == d.BuildDomainName() {
			if data.Editable == "0" {
				return nil, ErrRecordNotEditable
			}
			oldIP = net.ParseIP(data.Value)
			if ip.Equal(oldIP) { // success, nothing to change
				return ip, nil
			}
			break
		}
	}
	if oldIP != nil { // Found editable record with a different IP address, so remove it
		if err := d.removeRecord(ctx, client, oldIP); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrRemoveRecord, err)
		}
	}
	if err := d.createRecord(ctx, client, ip); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCreateRecord, err)
	}

	return ip, nil
}

type (
	dreamHostRecords struct {
		Result string `json:"result"`
		Data   []struct {
			Editable string `json:"editable"`
			Type     string `json:"type"`
			Record   string `json:"record"`
			Value    string `json:"value"`
		} `json:"data"`
	}
	dreamhostReponse struct {
		Result string `json:"result"`
		Data   string `json:"data"`
	}
)

func (d *dreamhost) defaultURLValues() (values url.Values) {
	uuid := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, uuid)
	//nolint:gomnd
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	//nolint:gomnd
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	values = url.Values{}
	values.Set("key", d.key)
	values.Set("unique_id", string(uuid))
	values.Set("format", "json")
	return values
}

func (d *dreamhost) getRecords(ctx context.Context, client *http.Client) (
	records dreamHostRecords, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := d.defaultURLValues()
	values.Set("cmd", "dns-list_records")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return records, err
	}
	setUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return records, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return records, fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&records); err != nil {
		return records, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	if records.Result != success {
		return records, fmt.Errorf("%w: %s", ErrUnsuccessfulResponse, records.Result)
	}
	return records, nil
}

func (d *dreamhost) removeRecord(ctx context.Context, client *http.Client, ip net.IP) error {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := d.defaultURLValues()
	values.Set("cmd", "dns-remove_record")
	values.Set("record", d.domain)
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	setUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	var dhResponse dreamhostReponse
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&dhResponse); err != nil {
		return fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	if dhResponse.Result != success { // this should not happen
		return fmt.Errorf("%w: %s - %s",
			ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func (d *dreamhost) createRecord(ctx context.Context, client *http.Client, ip net.IP) error {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := d.defaultURLValues()
	values.Set("cmd", "dns-add_record")
	values.Set("record", d.domain)
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	setUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			ErrBadHTTPStatus, response.StatusCode, bodyToSingleLine(response.Body))
	}

	var dhResponse dreamhostReponse
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&dhResponse); err != nil {
		return fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	if dhResponse.Result != success {
		return fmt.Errorf("%w: %s - %s",
			ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}
