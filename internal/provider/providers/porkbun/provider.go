package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
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
	domain       string
	host         string
	ttl          uint
	ipVersion    ipversion.IPVersion
	apiKey       string
	secretAPIKey string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		SecretAPIKey string `json:"secret_api_key"`
		APIKey       string `json:"api_key"`
		TTL          uint   `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:       domain,
		host:         host,
		ipVersion:    ipVersion,
		secretAPIKey: extraSettings.SecretAPIKey,
		apiKey:       extraSettings.APIKey,
		ttl:          extraSettings.TTL,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.apiKey == "":
		return fmt.Errorf("%w", errors.ErrEmptyAPIKey)
	case p.secretAPIKey == "":
		return fmt.Errorf("%w", errors.ErrEmptyAPISecret)
	}
	return nil
}

func (p *Provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Porkbun]", p.domain, p.host)
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
		Provider:  "<a href=\"https://www.porkbun.com/\">Porkbun DNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

func (p *Provider) getRecordIDs(ctx context.Context, client *http.Client, recordType string) (
	recordIDs []string, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/retrieveByNameType/" + p.domain + "/" + recordType + "/",
	}
	if p.host != "@" {
		u.Path += p.host
	}

	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	var responseData struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	for _, record := range responseData.Records {
		recordIDs = append(recordIDs, record.ID)
	}

	return recordIDs, nil
}

func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType string, ipStr string) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/create/" + p.domain,
	}
	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
		Content      string `json:"content"`
		Name         string `json:"name,omitempty"`
		Type         string `json:"type"`
		TTL          string `json:"ttl"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
		Content:      ipStr,
		Type:         recordType,
		Name:         p.host,
		TTL:          fmt.Sprint(p.ttl),
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}
	return nil
}

func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordType string, ipStr string, recordID string) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/edit/" + p.domain + "/" + recordID,
	}
	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
		Content      string `json:"content"`
		Type         string `json:"type"`
		TTL          string `json:"ttl"`
		Name         string `json:"name,omitempty"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
		Content:      ipStr,
		Type:         recordType,
		TTL:          fmt.Sprint(p.ttl),
		Name:         p.host,
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrRequestMarshal, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrBadRequest, err)
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("%w: %w", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}
	return nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	ipStr := ip.String()
	recordIDs, err := p.getRecordIDs(ctx, client, recordType)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(recordIDs) == 0 {
		err = p.createRecord(ctx, client, recordType, ipStr)
		if err != nil {
			return netip.Addr{}, err
		}
		return ip, nil
	}

	for _, recordID := range recordIDs {
		err = p.updateRecord(ctx, client, recordType, ipStr, recordID)
		if err != nil {
			return netip.Addr{}, err
		}
	}

	return ip, nil
}
