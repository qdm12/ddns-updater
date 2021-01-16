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
	"github.com/qdm12/golibs/network"
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

func (d *dreamhost) Update(ctx context.Context, client network.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}
	records, err := listDreamhostRecords(ctx, client, d.key)
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
		if err := removeDreamhostRecord(ctx, client, d.key, d.domain, oldIP); err != nil {
			return nil, fmt.Errorf("%w: %s", ErrRemoveRecord, err)
		}
	}
	if err := addDreamhostRecord(ctx, client, d.key, d.domain, ip); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrAddRecord, err)
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

func makeDreamhostDefaultValues(key string) (values url.Values) {
	uuid := make([]byte, 16)
	_, _ = io.ReadFull(rand.Reader, uuid)
	//nolint:gomnd
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // Version 4
	//nolint:gomnd
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // Variant is 10
	values = url.Values{}
	values.Set("key", key)
	values.Set("unique_id", string(uuid))
	values.Set("format", "json")
	return values
}

func listDreamhostRecords(ctx context.Context, client network.Client, key string) (
	records dreamHostRecords, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-list_records")
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return records, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return records, err
	} else if status != http.StatusOK {
		return records, fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
	}
	if err := json.Unmarshal(content, &records); err != nil {
		return records, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	} else if records.Result != success {
		return records, fmt.Errorf("%w: %s", ErrUnsuccessfulResponse, records.Result)
	}
	return records, nil
}

func removeDreamhostRecord(ctx context.Context, client network.Client,
	key, domain string, ip net.IP) error {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-remove_record")
	values.Set("record", domain)
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
	}
	var dhResponse dreamhostReponse
	if err := json.Unmarshal(content, &dhResponse); err != nil {
		return fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	} else if dhResponse.Result != success { // this should not happen
		return fmt.Errorf("%w: %s - %s",
			ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func addDreamhostRecord(ctx context.Context, client network.Client, key, domain string, ip net.IP) error {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-add_record")
	values.Set("record", domain)
	values.Set("type", recordType)
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
	}
	var dhResponse dreamhostReponse
	if err := json.Unmarshal(content, &dhResponse); err != nil {
		return fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	} else if dhResponse.Result != success {
		return fmt.Errorf("%w: %s - %s",
			ErrUnsuccessfulResponse, dhResponse.Result, dhResponse.Data)
	}
	return nil
}
