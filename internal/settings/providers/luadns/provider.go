package luadns

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	email     string
	token     string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		email:     extraSettings.Email,
		token:     extraSettings.Token,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case !verification.NewRegex().MatchEmail(p.email):
		return errors.ErrMalformedEmail
	case len(p.token) == 0:
		return errors.ErrEmptyToken
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.LuaDNS, p.ipVersion)
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
		Provider:  "<a href=\"https://www.luadns.com/\">LuaDNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
}

// Using https://www.luadns.com/api.html
func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	zoneID, err := p.getZoneID(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrGetZoneID, err)
	}

	record, err := p.getRecord(ctx, client, zoneID, ip)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrGetRecordInZone, err)
	}

	newRecord := record
	newRecord.Content = ip.String()
	if err := p.updateRecord(ctx, client, zoneID, newRecord); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUpdateRecord, err)
	}
	return ip, nil
}

type luaDNSRecord struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Content string `json:"content"`
	TTL     int    `json:"ttl"`
}

type luaDNSError struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func (p *provider) getZoneID(ctx context.Context, client *http.Client) (zoneID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   "/v1/zones",
		User:   url.UserPassword(p.email, p.token),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmaip.com")

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
		var errorObj luaDNSError
		if jsonErr := json.Unmarshal(b, &errorObj); jsonErr != nil {
			return 0, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
		}
		return 0, fmt.Errorf("%w: %s: %s", err, errorObj.Status, errorObj.Message)
	}
	type zone struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var zones []zone

	if err := json.Unmarshal(b, &zones); err != nil {
		return 0, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	for _, zone := range zones {
		if zone.Name == p.domain {
			return zone.ID, nil
		}
	}
	return 0, errors.ErrZoneNotFound
}

func (p *provider) getRecord(ctx context.Context, client *http.Client, zoneID int, ip net.IP) (
	record luaDNSRecord, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   fmt.Sprintf("/v1/zones/%d/records", zoneID),
		User:   url.UserPassword(p.email, p.token),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return record, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return record, err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return record, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
		var errorObj luaDNSError
		if jsonErr := json.Unmarshal(b, &errorObj); jsonErr != nil {
			return record, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
		}
		return record, fmt.Errorf("%w: %s: %s",
			err, errorObj.Status, errorObj.Message)
	}
	var records []luaDNSRecord

	if err := json.Unmarshal(b, &records); err != nil {
		return record, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}
	for _, record := range records {
		if record.Type == recordType && record.Name == utils.BuildURLQueryHostname(p.host, p.domain) {
			return record, nil
		}
	}
	return record, fmt.Errorf("%w: %s record in zone %d",
		errors.ErrRecordNotFound, recordType, zoneID)
}

func (p *provider) updateRecord(ctx context.Context, client *http.Client,
	zoneID int, newRecord luaDNSRecord) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   fmt.Sprintf("/v1/zones/%d/records/%d", zoneID, newRecord.ID),
		User:   url.UserPassword(p.email, p.token),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(newRecord); err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
		var errorObj luaDNSError
		if jsonErr := json.Unmarshal(b, &errorObj); jsonErr != nil {
			return fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
		}
		return fmt.Errorf("%w: %s: %s",
			err, errorObj.Status, errorObj.Message)
	}

	var updatedRecord luaDNSRecord
	if jsonErr := json.Unmarshal(b, &updatedRecord); jsonErr != nil {
		return fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if updatedRecord.Content != newRecord.Content {
		return fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, updatedRecord.Content)
	}
	return nil
}
