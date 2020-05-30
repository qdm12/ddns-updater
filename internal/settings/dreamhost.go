package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network"
)

type dreamhost struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	key       string
}

func NewDreamhost(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
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
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *dreamhost) isValid() error {
	switch {
	case !constants.MatchDreamhostKey(d.key):
		return fmt.Errorf("invalid key format")
	case d.host != "@":
		return fmt.Errorf(`host can only be "@"`)
	}
	return nil
}

func (d *dreamhost) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Dreamhost]", d.domain, d.host)
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

const success = "success"

func (d *dreamhost) Update(client network.Client, ip net.IP) (newIP net.IP, err error) {
	if ip == nil {
		return nil, fmt.Errorf("IP address was not given to updater")
	}
	records, err := listDreamhostRecords(client, d.key)
	if err != nil {
		return nil, err
	}
	var oldIP net.IP
	for _, data := range records.Data {
		if data.Type == "A" && data.Record == d.BuildDomainName() {
			if data.Editable == "0" {
				return nil, fmt.Errorf("record data is not editable")
			}
			oldIP = net.ParseIP(data.Value)
			if ip.Equal(oldIP) { // success, nothing to change
				return ip, nil
			}
			break
		}
	}
	if oldIP != nil { // Found editable record with a different IP address, so remove it
		if err := removeDreamhostRecord(client, d.key, d.domain, oldIP); err != nil {
			return nil, err
		}
	}
	return ip, addDreamhostRecord(client, d.key, d.domain, ip)
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

func makeDreamhostDefaultValues(key string) (values url.Values) { //nolint:unparam
	values.Set("key", key)
	values.Set("unique_id", uuid.New().String())
	values.Set("format", "json")
	return values
}

func listDreamhostRecords(client network.Client, key string) (records dreamHostRecords, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-list_records")
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return records, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return records, err
	} else if status != http.StatusOK {
		return records, fmt.Errorf("HTTP status %d", status)
	}
	if err := json.Unmarshal(content, &records); err != nil {
		return records, err
	} else if records.Result != success {
		return records, fmt.Errorf(records.Result)
	}
	return records, nil
}

func removeDreamhostRecord(client network.Client, key, domain string, ip net.IP) error { //nolint:dupl
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-remove_record")
	values.Set("record", domain)
	values.Set("type", "A")
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	var dhResponse dreamhostReponse
	if err := json.Unmarshal(content, &dhResponse); err != nil {
		return err
	} else if dhResponse.Result != success { // this should not happen
		return fmt.Errorf("%s - %s", dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func addDreamhostRecord(client network.Client, key, domain string, ip net.IP) error { //nolint:dupl
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-add_record")
	values.Set("record", domain)
	values.Set("type", "A")
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	var dhResponse dreamhostReponse
	if err := json.Unmarshal(content, &dhResponse); err != nil {
		return err
	} else if dhResponse.Result != success {
		return fmt.Errorf("%s - %s", dhResponse.Result, dhResponse.Data)
	}
	return nil
}
