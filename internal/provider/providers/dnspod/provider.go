package dnspod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	token     string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	if p.token == "" {
		return fmt.Errorf("%w", errors.ErrEmptyToken)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.DNSPod, p.ipVersion)
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
		Provider:  "<a href=\"https://www.dnspod.cn/\">DNSPod</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetAccept(request, "application/json")
	headers.SetUserAgent(request)
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	u := url.URL{
		Scheme: "https",
		Host:   "dnsapi.cn",
		Path:   "/Record.List",
	}

	values := url.Values{}
	values.Set("login_token", p.token)
	values.Set("format", "json")
	values.Set("domain", p.domain)
	values.Set("length", "200")
	values.Set("sub_domain", p.host)
	values.Set("record_type", recordType)
	encodedValues := values.Encode()
	buffer := bytes.NewBufferString(encodedValues)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
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
	err = decoder.Decode(&recordResp)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	var recordID, recordLine string
	for _, record := range recordResp.Records {
		if record.Type == recordType && record.Name == p.host {
			receivedIP, err := netip.ParseAddr(record.Value)
			if err == nil && ip.Compare(receivedIP) == 0 {
				return ip, nil
			}
			recordID = record.ID
			recordLine = record.Line
			break
		}
	}
	if recordID == "" {
		return netip.Addr{}, fmt.Errorf("%w", errors.ErrNotFound)
	}

	u.Path = "/Record.Ddns"
	values = url.Values{}
	values.Set("login_token", p.token)
	values.Set("format", "json")
	values.Set("domain", p.domain)
	values.Set("record_id", recordID)
	values.Set("value", ip.String())
	values.Set("record_line", recordLine)
	values.Set("sub_domain", p.host)
	encodedValues = values.Encode()
	buffer = bytes.NewBufferString(encodedValues)

	request, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, err
	}
	p.setHeaders(request)

	response, err = client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return netip.Addr{}, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("reading response body: %w", err)
	}

	var ddnsResp struct {
		Record struct {
			ID    int64  `json:"id"`
			Value string `json:"value"`
			Name  string `json:"name"`
		} `json:"record"`
	}
	err = json.Unmarshal(data, &ddnsResp)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	ipStr := ddnsResp.Record.Value
	receivedIP, err := netip.ParseAddr(ipStr)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("%w: %w", errors.ErrIPReceivedMalformed, err)
	} else if ip.Compare(receivedIP) != 0 {
		return netip.Addr{}, fmt.Errorf("%w: sent ip %s to update but received %s",
			errors.ErrIPReceivedMismatch, ip, newIP)
	}
	return ip, nil
}
