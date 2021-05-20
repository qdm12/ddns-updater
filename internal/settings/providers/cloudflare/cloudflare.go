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
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type cloudflare struct {
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
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (c *cloudflare, err error) {
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
	c = &cloudflare{
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
	}
	if err := c.isValid(); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *cloudflare) isValid() error {
	switch {
	case len(c.key) > 0: // email and key must be provided
		switch {
		case !c.matcher.CloudflareKey(c.key):
			return errors.ErrMalformedKey
		case !verification.NewVerifier().MatchEmail(c.email):
			return errors.ErrMalformedEmail
		}
	case len(c.userServiceKey) > 0: // only user service key
		if !c.matcher.CloudflareKey(c.key) {
			return errors.ErrMalformedUserServiceKey
		}
	default: // constants.API token only
	}
	switch {
	case len(c.zoneIdentifier) == 0:
		return errors.ErrEmptyZoneIdentifier
	case c.ttl == 0:
		return errors.ErrEmptyTTL
	}
	return nil
}

func (c *cloudflare) String() string {
	return utils.ToString(c.domain, c.host, constants.Cloudflare, c.ipVersion)
}

func (c *cloudflare) Domain() string {
	return c.domain
}

func (c *cloudflare) Host() string {
	return c.host
}

func (c *cloudflare) IPVersion() ipversion.IPVersion {
	return c.ipVersion
}

func (c *cloudflare) Proxied() bool {
	return c.proxied
}

func (c *cloudflare) BuildDomainName() string {
	return utils.BuildDomainName(c.host, c.domain)
}

func (c *cloudflare) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", c.BuildDomainName(), c.BuildDomainName())),
		Host:      models.HTML(c.Host()),
		Provider:  "<a href=\"https://www.cloudflare.com\">Cloudflare</a>",
		IPVersion: models.HTML(c.ipVersion.String()),
	}
}

func (c *cloudflare) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	switch {
	case len(c.token) > 0:
		headers.SetAuthBearer(request, c.token)
	case len(c.userServiceKey) > 0:
		request.Header.Set("X-Auth-User-Service-Key", c.userServiceKey)
	case len(c.email) > 0 && len(c.key) > 0:
		request.Header.Set("X-Auth-Email", c.email)
		request.Header.Set("X-Auth-Key", c.key)
	}
}

// Obtain domain ID.
// See https://api.cloudflare.com/#dns-records-for-a-zone-list-dns-records.
func (c *cloudflare) getRecordID(ctx context.Context, client *http.Client, newIP net.IP) (
	identifier string, upToDate bool, err error) {
	recordType := constants.A
	if newIP.To4() == nil {
		recordType = constants.AAAA
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.cloudflare.com",
		Path:   fmt.Sprintf("/client/v4/zones/%s/dns_records", c.zoneIdentifier),
	}
	values := url.Values{}
	values.Set("type", recordType)
	values.Set("name", c.BuildDomainName())
	values.Set("page", "1")
	values.Set("per_page", "1")
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", false, err
	}
	c.setHeaders(request)

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

func (c *cloudflare) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}
	identifier, upToDate, err := c.getRecordID(ctx, client, ip)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errors.ErrGetRecordID, err)
	} else if upToDate {
		return ip, nil
	}

	u := url.URL{
		Scheme: "https",
		Host:   "api.cloudflare.com",
		Path:   fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", c.zoneIdentifier, identifier),
	}

	requestData := struct {
		Type    string `json:"type"`    // constants.A or constants.AAAA depending on ip address given
		Name    string `json:"name"`    // DNS record name i.e. example.com
		Content string `json:"content"` // ip address
		Proxied bool   `json:"proxied"` // whether the record is receiving the performance and security benefits of Cloudflare
		TTL     uint   `json:"ttl"`
	}{
		Type:    recordType,
		Name:    c.BuildDomainName(),
		Content: ip.String(),
		Proxied: c.proxied,
		TTL:     c.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(requestData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrRequestEncode, err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPut, u.String(), buffer)
	if err != nil {
		return nil, err
	}

	c.setHeaders(request)

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
