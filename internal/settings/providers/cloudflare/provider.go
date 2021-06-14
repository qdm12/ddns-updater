package cloudflare

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
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type provider struct {
	domain         string
	host           string
	ipVersion      ipversion.IPVersion
	key            string
	token          string
	email          string
	userServiceKey string
	zoneIdentifier string
	proxied        bool
	ttl            uint
	matcher        regex.Matcher
	logger         log.Logger
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Key            string `json:"key"`
		Token          string `json:"token"`
		Email          string `json:"email"`
		UserServiceKey string `json:"user_service_key"`
		ZoneIdentifier string `json:"zone_identifier"`
		Proxied        bool   `json:"proxied"`
		TTL            uint   `json:"ttl"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		key:            extraSettings.Key,
		token:          extraSettings.Token,
		email:          extraSettings.Email,
		userServiceKey: extraSettings.UserServiceKey,
		zoneIdentifier: extraSettings.ZoneIdentifier,
		proxied:        extraSettings.Proxied,
		ttl:            extraSettings.TTL,
		matcher:        matcher,
		logger:         logger,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case len(p.key) > 0: // email and key must be provided
		switch {
		case !p.matcher.CloudflareKey(p.key):
			return errors.ErrMalformedKey
		case !verification.NewVerifier().MatchEmail(p.email):
			return errors.ErrMalformedEmail
		}
	case len(p.userServiceKey) > 0: // only user service key
		if !p.matcher.CloudflareKey(p.key) {
			return errors.ErrMalformedUserServiceKey
		}
	default: // constants.API token only
	}
	switch {
	case len(p.zoneIdentifier) == 0:
		return errors.ErrEmptyZoneIdentifier
	case p.ttl == 0:
		return errors.ErrEmptyTTL
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Cloudflare, p.ipVersion)
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
	return p.proxied
}

func (p *provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.cloudflare.com\">Cloudflare</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	switch {
	case len(p.token) > 0:
		headers.SetAuthBearer(request, p.token)
	case len(p.userServiceKey) > 0:
		request.Header.Set("X-Auth-User-Service-Key", p.userServiceKey)
	case len(p.email) > 0 && len(p.key) > 0:
		request.Header.Set("X-Auth-Email", p.email)
		request.Header.Set("X-Auth-Key", p.key)
	}
}

// Obtain domain ID.
// See https://api.cloudflare.com/#dns-records-for-a-zone-list-dns-records.
func (p *provider) getRecordID(ctx context.Context, client *http.Client, newIP net.IP) (
	identifier string, upToDate bool, err error) {
	recordType := constants.A
	if newIP.To4() == nil {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.cloudflare.com",
		Path:   fmt.Sprintf("/client/v4/zones/%s/dns_records", p.zoneIdentifier),
	}
	values := url.Values{}
	values.Set("type", recordType)
	values.Set("name", utils.BuildURLQueryHostname(p.host, p.domain))
	values.Set("page", "1")
	values.Set("per_page", "1")
	u.RawQuery = values.Encode()

	p.logger.Debug("HTTP GET: " + u.String())

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", false, err
	}
	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return "", false, err
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", false, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	listRecordsResponse := struct {
		Success bool     `json:"success"`
		Errors  []string `json:"errors"`
		Result  []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
		} `json:"result"`
	}{}
	if err := decoder.Decode(&listRecordsResponse); err != nil {
		return "", false, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	switch {
	case len(listRecordsResponse.Errors) > 0:
		return "", false, fmt.Errorf("%w: %s",
			errors.ErrUnsuccessfulResponse, strings.Join(listRecordsResponse.Errors, ","))
	case !listRecordsResponse.Success:
		return "", false, errors.ErrUnsuccessfulResponse
	case len(listRecordsResponse.Result) == 0:
		return "", false, errors.ErrNoResultReceived
	case len(listRecordsResponse.Result) > 1:
		return "", false, fmt.Errorf("%w: %d instead of 1",
			errors.ErrNumberOfResultsReceived, len(listRecordsResponse.Result))
	case listRecordsResponse.Result[0].Content == newIP.String(): // up to date
		return "", true, nil
	}
	return listRecordsResponse.Result[0].ID, false, nil
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}
	identifier, upToDate, err := p.getRecordID(ctx, client, ip)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errors.ErrGetRecordID, err)
	} else if upToDate {
		return ip, nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.cloudflare.com",
		Path:   fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", p.zoneIdentifier, identifier),
	}

	requestData := struct {
		Type    string `json:"type"`    // constants.A or constants.AAAA depending on ip address given
		Name    string `json:"name"`    // DNS record name i.e. example.com
		Content string `json:"content"` // ip address
		Proxied bool   `json:"proxied"` // whether the record is receiving the performance and security benefits of Cloudflare
		TTL     uint   `json:"ttl"`
	}{
		Type:    recordType,
		Name:    utils.BuildURLQueryHostname(p.host, p.domain),
		Content: ip.String(),
		Proxied: p.proxied,
		TTL:     p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(requestData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrRequestEncode, err)
	}

	p.logger.Debug("HTTP PUT: " + u.String() + ": " + buffer.String())

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return nil, err
	}

	p.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode > http.StatusUnsupportedMediaType {
		return nil, fmt.Errorf("%w: %d: %s",
			errors.ErrBadHTTPStatus, response.StatusCode, utils.BodyToSingleLine(response.Body))
	}

	decoder := json.NewDecoder(response.Body)
	var parsedJSON struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			Content string `json:"content"`
		} `json:"result"`
	}
	if err := decoder.Decode(&parsedJSON); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if !parsedJSON.Success {
		var errStr string
		for _, e := range parsedJSON.Errors {
			errStr += fmt.Sprintf("error %d: %s; ", e.Code, e.Message)
		}
		return nil, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, errStr)
	}

	newIP = net.ParseIP(parsedJSON.Result.Content)
	if newIP == nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMalformed, parsedJSON.Result.Content)
	} else if !newIP.Equal(ip) {
		return nil, fmt.Errorf("%w: %s", errors.ErrIPReceivedMismatch, newIP.String())
	}
	return newIP, nil
}
