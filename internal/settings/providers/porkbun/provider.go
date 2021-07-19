package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain       string
	host         string
	ttl          uint
	ipVersion    ipversion.IPVersion
	apiKey       string
	secretApiKey string
	logger       log.Logger
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		SecretApiKey string `json:"secret_api_key"`
		ApiKey       string `json:"api_key"`
		Name         bool   `json:"name"`
		Type         string `json:"type"`
		Content      string `json:"content"`
		TTL          uint   `json:"ttl"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:       domain,
		host:         host,
		ipVersion:    ipVersion,
		secretApiKey: extraSettings.SecretApiKey,
		apiKey:       extraSettings.ApiKey,
		ttl:          extraSettings.TTL,
		logger:       logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case p.apiKey == "":
		return errors.ErrEmptyAppKey
	case p.secretApiKey == "":
		return errors.ErrEmptyConsumerKey
	}
	return nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Porkbun]", p.domain, p.host)
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
		Provider:  "<a href=\"https://www.porkbun.com/\">Porkbun DNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

func (p *provider) getRecords(ctx context.Context, client *http.Client) (recordIDs []string, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/retrieve/" + p.domain,
	}
	postRecordsParams := struct {
		SecretApiKey string `json:"secretapikey"`
		ApiKey       string `json:"apikey"`
	}{
		SecretApiKey: p.secretApiKey,
		ApiKey:       p.apiKey,
	}
	bodyBytes, err := json.Marshal(postRecordsParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrRequestMarshal, err)
	}

	p.logger.Debug("HTTP POST getRecords: " + u.String() + ": " + string(bodyBytes))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, err)
	}

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	type domainRecords struct {
		Status  string `json:"status"`
		Records []struct {
			Id      string `json:"id"`
			Name    string `json:"title"`
			Type    string `json:"type"`
			Content string `json:"content"`
			TTL     string `json:"ttl"`
			Prio    string `json:"prio"`
			Notes   string `json:"notes"`
		} `json:"records"`
	}
	defer response.Body.Close()
	var responseData domainRecords
	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&responseData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	var _recordIDs []string
	for _, recordID := range responseData.Records {
		if strings.HasSuffix(recordID.Content, p.domain) {
			_recordIDs = append(_recordIDs, recordID.Id)
		}
	}
	p.logger.Debug("getRecords: " + strings.Join(_recordIDs, ", "))
	return _recordIDs, nil
}

func (p *provider) createRecord(ctx context.Context, client *http.Client,
	recordType string, ipStr string) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   fmt.Sprintf("/api/json/v3/dns/create/%s", p.domain),
	}
	postRecordsParams := struct {
		SecretApiKey string `json:"secretapikey"`
		ApiKey       string `json:"apikey"`
		Content      string `json:"content"`
		Name         string `json:"name,omitempty"`
		Type         string `json:"type"`
		TTL          string `json:"ttl"`
	}{
		SecretApiKey: p.secretApiKey,
		ApiKey:       p.apiKey,
		Content:      ipStr,
		Type:         recordType,
		Name:         p.host,
		TTL:          fmt.Sprint(p.ttl),
	}
	bodyBytes, err := json.Marshal(postRecordsParams)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrRequestMarshal, err)
	}

	p.logger.Debug("HTTP POST createRecord: " + u.String() + ": " + string(bodyBytes))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}
	return nil
}

func (p *provider) updateRecord(ctx context.Context, client *http.Client,
	recordType string, ipStr string, recordID string) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   fmt.Sprintf("/api/json/v3/dns/edit/%s/%s", p.domain, recordID),
	}
	postRecordsParams := struct {
		SecretApiKey string `json:"secretapikey"`
		ApiKey       string `json:"apikey"`
		Content      string `json:"content"`
		Type         string `json:"type"`
		TTL          string `json:"ttl"`
		Name         string `json:"name,omitempty"`
	}{
		SecretApiKey: p.secretApiKey,
		ApiKey:       p.apiKey,
		Content:      ipStr,
		Type:         recordType,
		TTL:          fmt.Sprint(p.ttl),
		Name:         p.host,
	}
	bodyBytes, err := json.Marshal(postRecordsParams)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrRequestMarshal, err)
	}

	p.logger.Debug("HTTP POST updateRecord: " + u.String() + ": " + string(bodyBytes))

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewBuffer(bodyBytes))
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrBadRequest, err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}
	return nil
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	var ipStr string
	if ip.To4() == nil { // IPv6
		recordType = constants.AAAA
		ipStr = ip.To16().String()
	} else {
		ipStr = ip.To4().String()
	}
	recordIDs, err := p.getRecords(ctx, client)
	if err != nil {
		return nil, err
	}
	if len(recordIDs) == 0 {
		if err := p.createRecord(ctx, client, recordType, ipStr); err != nil {
			return nil, err
		}
	} else {
		for _, recordID := range recordIDs {
			if err := p.updateRecord(ctx, client, recordType, ipStr, recordID); err != nil {
				return nil, err
			}
		}
	}

	return ip, nil
}
