package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	netlib "github.com/qdm12/golibs/network"
)

type linode struct {
	domain    string
	host      string
	ipVersion models.IPVersion
	token     string
}

func NewLinode(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	_ bool, _ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		Token string `json:"token"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	l := &linode{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		token:     extraSettings.Token,
	}
	if err := l.isValid(); err != nil {
		return nil, err
	}
	return l, nil
}

func (l *linode) isValid() error {
	if len(l.token) == 0 {
		return ErrEmptyToken
	}
	return nil
}

func (l *linode) String() string {
	return toString(l.domain, l.host, constants.LUADNS, l.ipVersion)
}

func (l *linode) Domain() string {
	return l.domain
}

func (l *linode) Host() string {
	return l.host
}

func (l *linode) DNSLookup() bool {
	return true
}

func (l *linode) IPVersion() models.IPVersion {
	return l.ipVersion
}

func (l *linode) BuildDomainName() string {
	return buildDomainName(l.host, l.domain)
}

func (l *linode) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", l.BuildDomainName(), l.BuildDomainName())),
		Host:      models.HTML(l.Host()),
		Provider:  "<a href=\"https://cloud.linode.com/\">Linode</a>",
		IPVersion: models.HTML(l.ipVersion),
	}
}

// Using https://www.linode.com/docs/api/domains/
func (l *linode) Update(ctx context.Context, client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	domainID, err := l.getDomainID(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrGetDomainID, err)
	}

	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}

	recordID, err := l.getRecordID(ctx, client, domainID, recordType)
	if errors.Is(err, ErrNotFound) {
		err := l.createRecord(ctx, client, domainID, recordType, ip)
		if err != nil {
			return nil, fmt.Errorf("%w: %s", ErrCreateRecord, err)
		}
		return ip, nil
	} else if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrGetRecordID, err)
	}

	if err := l.updateRecord(ctx, client, domainID, recordID, ip); err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUpdateRecord, err)
	}

	return ip, nil
}

type linodeError struct {
	Field  string `json:"field"`
	Reason string `json:"reason"`
}

func (l *linode) getDomainID(ctx context.Context, client netlib.Client) (domainID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   "/v4/domains",
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+l.token)
	r.Header.Set("oauth", "domains:read_only")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	r.Header.Set("X-Filter", `{"domain": "`+l.domain+`"}`)

	content, status, err := client.Do(r)
	if err != nil {
		return 0, err
	}

	if status != http.StatusOK {
		err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
		var errorObj linodeError
		if jsonErr := json.Unmarshal(content, &errorObj); jsonErr != nil {
			return 0, fmt.Errorf("%w: %s", err, string(content))
		}
		return 0, fmt.Errorf("%w: %s: %s", err, errorObj.Field, errorObj.Reason)
	}

	var domains []struct {
		ID     *int   `json:"id,omitempty"`
		Type   string `json:"type"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(content, &domains); err != nil {
		return 0, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	switch len(domains) {
	case 0:
		return 0, ErrNotFound
	case 1:
	default:
		return 0, fmt.Errorf("%w: %d records instead of 1",
			ErrNumberOfResultsReceived, len(domains))
	}

	if domains[0].Status == "disabled" {
		return 0, ErrDomainDisabled
	}

	if domains[0].ID == nil {
		return 0, ErrDomainIDNotFound
	}

	return *domains[0].ID, nil
}

func (l *linode) getRecordID(ctx context.Context, client netlib.Client,
	domainID int, recordType string) (recordID int, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   "/v4/domains/" + strconv.Itoa(domainID) + "/records",
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+l.token)
	r.Header.Set("oauth", "domains:read_only")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

	content, status, err := client.Do(r)
	if err != nil {
		return 0, err
	}

	if status != http.StatusOK {
		err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
		var errorObj linodeError
		if jsonErr := json.Unmarshal(content, &errorObj); jsonErr != nil {
			return 0, fmt.Errorf("%w: %s", err, string(content))
		}
		return 0, fmt.Errorf("%w: %s: %s", err, errorObj.Field, errorObj.Reason)
	}

	var domainRecords []struct {
		ID   int    `json:"id"`
		Type string `json:"type"`
	}
	if err := json.Unmarshal(content, &domainRecords); err != nil {
		return 0, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}

	for _, domainRecord := range domainRecords {
		if domainRecord.Type == recordType {
			return domainRecord.ID, nil
		}
	}

	return 0, ErrNotFound
}

func (l *linode) createRecord(ctx context.Context, client netlib.Client,
	domainID int, recordType string, ip net.IP) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   "/v4/domains/" + strconv.Itoa(domainID) + "/records",
	}

	data := struct {
		Type string `json:"type"`
		Host string `json:"name"`
		IP   string `json:"target"`
	}{
		Type: recordType,
		Host: l.host,
		IP:   ip.String(),
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("%w: %s", ErrRequestMarshal, err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), buffer)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+l.token)
	r.Header.Set("oauth", "domains:read_write")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

	content, status, err := client.Do(r)
	if err != nil {
		return err
	}

	if status == http.StatusOK {
		return nil
	}

	err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
	var errorObj linodeError
	if jsonErr := json.Unmarshal(content, &errorObj); jsonErr != nil {
		return fmt.Errorf("%w: %s", err, string(content))
	}
	return fmt.Errorf("%w: %s: %s", err, errorObj.Field, errorObj.Reason)
}

func (l *linode) updateRecord(ctx context.Context, client netlib.Client,
	domainID, recordID int, ip net.IP) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.linode.com",
		Path:   "/v4/domains/" + strconv.Itoa(domainID) + "/records/" + strconv.Itoa(recordID),
	}

	data := struct {
		IP string `json:"target"`
	}{
		IP: ip.String(),
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(data); err != nil {
		return fmt.Errorf("%w: %s", ErrRequestMarshal, err)
	}

	r, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer "+l.token)
	r.Header.Set("oauth", "domains:read_write")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")

	b, status, err := client.Do(r)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}

	err = fmt.Errorf("%w: %d", ErrBadHTTPStatus, status)
	var errorObj linodeError
	if err := json.Unmarshal(b, &errorObj); err != nil {
		return fmt.Errorf("%w: %s", err, string(b))
	}
	return fmt.Errorf("%w: %s: %s", err, errorObj.Field, errorObj.Reason)
}
