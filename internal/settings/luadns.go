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
	"github.com/qdm12/golibs/verification"
)

type luaDNS struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	email     string
	token     string
}

func NewLuaDNS(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	_ bool, _ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Email string `json:"email"`
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	l := &luaDNS{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		email:     extraSettings.Email,
		token:     extraSettings.Token,
	}
	if err := l.isValid(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *luaDNS) isValid() error {
	switch {
	case !verification.NewRegex().MatchEmail(l.email):
		return ErrMalformedEmail
	case len(l.token) == 0:
		return ErrEmptyToken
	}
	return nil
}

func (l *luaDNS) String() string {
	return toString(l.domain, l.host, constants.LUADNS, l.ipVersion)
}

func (l *luaDNS) Domain() string {
	return l.domain
}

func (l *luaDNS) Host() string {
	return l.host
}

func (l *luaDNS) DNSLookup() bool {
	return true
}

func (l *luaDNS) IPVersion() models.IPVersion {
	return l.ipVersion
}

func (l *luaDNS) BuildDomainName() string {
	return buildDomainName(l.host, l.domain)
}

func (l *luaDNS) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", l.BuildDomainName(), l.BuildDomainName())),
		Host:      models.HTML(l.Host()),
		Provider:  "<a href=\"https://www.luadns.com/\">LuaDNS</a>",
		IPVersion: models.HTML(l.ipVersion),
	}
}

// Using https://www.luadns.com/api.html
func (l *luaDNS) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	zoneID, err := l.getZoneID(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrGetZoneID, err)
	}

	record, err := l.getRecord(ctx, client, zoneID, ip)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrGetRecordInZone, err)
	}

	newRecord := record
	newRecord.Content = ip.String()
	if err := l.updateRecord(ctx, client, zoneID, newRecord); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpdateRecord, err)
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

func (l *luaDNS) getZoneID(ctx context.Context, client *http.Client) (zoneID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   "/v1/zones",
		User:   url.UserPassword(l.email, l.token),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

	response, err := client.Do(request)
	if err != nil {
		return 0, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, response.StatusCode)
		var errorObj luaDNSError
		decoder := json.NewDecoder(response.Body)
		if jsonErr := decoder.Decode(&errorObj); jsonErr != nil {
			return 0, err
		}
		return 0, fmt.Errorf("%w: %s: %s", err, errorObj.Status, errorObj.Message)
	}
	type zone struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	var zones []zone

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&zones); err != nil {
		return 0, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	for _, zone := range zones {
		if zone.Name == l.domain {
			return zone.ID, nil
		}
	}
	return 0, ErrZoneNotFound
}

func (l *luaDNS) getRecord(ctx context.Context, client *http.Client, zoneID int, ip net.IP) (
	record luaDNSRecord, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   fmt.Sprintf("/v1/zones/%d/records", zoneID),
		User:   url.UserPassword(l.email, l.token),
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return record, err
	}
	request.Header.Set("Accept", "application/json")
	request.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

	response, err := client.Do(request)
	if err != nil {
		return record, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, response.StatusCode)
		var errorObj luaDNSError

		decoder := json.NewDecoder(response.Body)
		if jsonErr := decoder.Decode(&errorObj); jsonErr != nil {
			return record, err
		}
		return record, fmt.Errorf("%w: %s: %s",
			err, errorObj.Status, errorObj.Message)
	}
	var records []luaDNSRecord

	decoder := json.NewDecoder(response.Body)
	if err := decoder.Decode(&records); err != nil {
		return record, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}
	for _, record := range records {
		if record.Type == recordType {
			return record, nil
		}
	}
	return record, fmt.Errorf("%w: %s record in zone %d",
		ErrRecordNotFound, recordType, zoneID)
}

func (l *luaDNS) updateRecord(ctx context.Context, client *http.Client,
	zoneID int, newRecord luaDNSRecord) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.luadns.com",
		Path:   fmt.Sprintf("/v1/zones/%d/records/%d", zoneID, newRecord.ID),
		User:   url.UserPassword(l.email, l.token),
	}
	data, err := json.Marshal(newRecord)
	if err != nil {
		return err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	request.Header.Set("Accept", "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, response.StatusCode)
		var errorObj luaDNSError
		decoder := json.NewDecoder(response.Body)
		if jsonErr := decoder.Decode(&errorObj); jsonErr != nil {
			return err
		}
		return fmt.Errorf("%w: %s: %s",
			err, errorObj.Status, errorObj.Message)
	}

	var updatedRecord luaDNSRecord
	decoder := json.NewDecoder(response.Body)
	if jsonErr := decoder.Decode(&updatedRecord); jsonErr != nil {
		return fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	if updatedRecord.Content != newRecord.Content {
		return fmt.Errorf("%w: %s", ErrIPReceivedMismatch, updatedRecord.Content)
	}
	return nil
}
