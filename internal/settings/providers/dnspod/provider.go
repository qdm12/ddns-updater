package dnspod

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	token     string
	logger    log.Logger
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
		logger:    logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if len(p.token) == 0 {
		return errors.ErrEmptyToken
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.DNSPod, p.ipVersion)
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
		Provider:  "<a href=\"https://www.dnspod.cn/\">DNSPod</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) setHeaders(request *http.Request) {
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetAccept(request, "application/json")
	headers.SetUserAgent(request)
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
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

	p.logger.Debug("HTTP POST: " + u.String() + ": " + encodedValues)

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
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
	if err := decoder.Decode(&recordResp); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	var recordID, recordLine string
	for _, record := range recordResp.Records {
		if record.Type == constants.A && record.Name == p.host {
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
		return nil, errors.ErrNotFound
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

	p.logger.Debug("HTTP POST: " + u.String() + ": " + encodedValues)

	request, err = http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	p.setHeaders(request)

	response, err = client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
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
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	ipStr := ddnsResp.Record.Value
	receivedIP := net.ParseIP(ipStr)
	if receivedIP == nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, ipStr)
	} else if !ip.Equal(receivedIP) {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, receivedIP.String())
	}
	return ip, nil
}
