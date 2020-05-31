package settings

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/network"
)

type dnspod struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	token     string
}

func NewDNSPod(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &dnspod{
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

func (d *dnspod) isValid() error {
	if len(d.token) == 0 {
		return fmt.Errorf("token cannot be empty")
	}
	return nil
}

func (d *dnspod) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: DNSPod]", d.domain, d.host)
}

func (d *dnspod) Domain() string {
	return d.domain
}

func (d *dnspod) Host() string {
	return d.host
}

func (d *dnspod) IPVersion() models.IPVersion {
	return d.ipVersion
}

func (d *dnspod) DNSLookup() bool {
	return d.dnsLookup
}

func (d *dnspod) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *dnspod) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://www.dnspod.cn/\">DNSPod</a>",
		IPVersion: models.HTML(d.ipVersion),
	}
}

func (d *dnspod) Update(client network.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}
	u := url.URL{
		Scheme: "https",
		Host:   "dnsapi.cn",
		Path:   "/Record.List",
	}
	values := url.Values{}
	values.Set("login_token", d.token)
	values.Set("format", "json")
	values.Set("domain", d.domain)
	values.Set("length", "200")
	values.Set("sub_domain", d.host)
	values.Set("record_type", recordType)
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	var recordResp struct {
		Records []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			Line  string `json:"line"`
		} `json:"records"`
	}
	if err := json.Unmarshal(content, &recordResp); err != nil {
		return nil, err
	}
	var recordID, recordLine string
	for _, record := range recordResp.Records {
		if record.Type == A && record.Name == d.host {
			receivedIP := net.ParseIP(record.Value)
			if ip.Equal(receivedIP) {
				return ip, nil
			}
			recordID = record.ID
			recordLine = record.Line
			break
		}
	}
	if len(recordID) == 0 {
		return nil, fmt.Errorf("record not found")
	}

	u.Path = "/Record.Ddns"
	values = url.Values{}
	values.Set("login_token", d.token)
	values.Set("format", "json")
	values.Set("domain", d.domain)
	values.Set("record_id", recordID)
	values.Set("value", ip.String())
	values.Set("record_line", recordLine)
	values.Set("sub_domain", d.host)
	u.RawQuery = values.Encode()
	r, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBufferString(values.Encode()))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err = client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
	var ddnsResp struct {
		Record struct {
			ID    int64  `json:"id"`
			Value string `json:"value"`
			Name  string `json:"name"`
		} `json:"record"`
	}
	if err := json.Unmarshal(content, &ddnsResp); err != nil {
		return nil, err
	}
	receivedIP := net.ParseIP(ddnsResp.Record.Value)
	if !ip.Equal(receivedIP) {
		return nil, fmt.Errorf("ip not set")
	}
	return ip, nil
}
