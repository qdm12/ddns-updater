package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
)

type dnspod struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	dnsLookup bool
	token     string
}

func NewDNSPod(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
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
		return ErrEmptyToken
	}
	return nil
}

func (d *dnspod) String() string {
	return toString(d.domain, d.host, constants.DNSPOD, d.ipVersion)
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

func (d *dnspod) setHeaders(request *http.Request) {
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
}

func (d *dnspod) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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
	buffer := bytes.NewBufferString(values.Encode())

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrBadHTTPStatus, response.StatusCode)
	}

	decoder := json.NewDecoder(response.Body)
	var recordResp struct {
		Records []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			Line  string `json:"line"`
		} `json:"records"`
	}
	if err := decoder.Decode(&recordResp); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
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
		return nil, ErrNotFound
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
	buffer = bytes.NewBufferString(values.Encode())

	request, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	d.setHeaders(request)

	response, err = client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d", ErrBadHTTPStatus, response.StatusCode)
	}

	decoder = json.NewDecoder(response.Body)
	var ddnsResp struct {
		Record struct {
			ID    int64  `json:"id"`
			Value string `json:"value"`
			Name  string `json:"name"`
		} `json:"record"`
	}
	if err := decoder.Decode(&ddnsResp); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	ipStr := ddnsResp.Record.Value
	receivedIP := net.ParseIP(ipStr)
	if receivedIP == nil {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMalformed, ipStr)
	} else if !ip.Equal(receivedIP) {
		return nil, fmt.Errorf("%w: %s", ErrIPReceivedMismatch, receivedIP.String())
	}
	return ip, nil
}
