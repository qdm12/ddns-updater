package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/qdm12/golibs/verification"
)

type Provider struct {
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
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		Key            string `json:"key"`
		Token          string `json:"token"`
		Email          string `json:"email"`
		UserServiceKey string `json:"user_service_key"`
		ZoneIdentifier string `json:"zone_identifier"`
		Proxied        bool   `json:"proxied"`
		TTL            uint   `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
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
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

var (
	keyRegex            = regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	userServiceKeyRegex = regexp.MustCompile(`^v1\.0.+$`)
)

func (p *Provider) isValid() error {
	switch {
	case p.key != "": // email and key must be provided
		switch {
		case !keyRegex.MatchString(p.key):
			return fmt.Errorf("%w", errors.ErrMalformedKey)
		case !verification.NewVerifier().MatchEmail(p.email):
			return fmt.Errorf("%w", errors.ErrMalformedEmail)
		}
	case p.userServiceKey != "": // only user service key
		if !userServiceKeyRegex.MatchString(p.userServiceKey) {
			return fmt.Errorf("%w", errors.ErrMalformedUserServiceKey)
		}
	default: // constants.API token only
	}
	switch {
	case p.zoneIdentifier == "":
		return fmt.Errorf("%w", errors.ErrEmptyZoneIdentifier)
	case p.ttl == 0:
		return fmt.Errorf("%w", errors.ErrEmptyTTL)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Cloudflare, p.ipVersion)
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
	return p.proxied
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://www.cloudflare.com\">Cloudflare</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *Provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	switch {
	case p.token != "":
		headers.SetAuthBearer(request, p.token)
	case p.userServiceKey != "":
		request.Header.Set("X-Auth-User-Service-Key", p.userServiceKey)
	case p.email != "" && p.key != "":
		request.Header.Set("X-Auth-Email", p.email)
		request.Header.Set("X-Auth-Key", p.key)
	}
}

// Obtain domain ID.
// See https://api.cloudflare.com/#dns-records-for-a-zone-list-dns-records.
func (p *Provider) getRecordID(ctx context.Context, client *http.Client, newIP net.IP) (
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
	err = decoder.Decode(&listRecordsResponse)
	if err != nil {
		return "", false, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
	}

	switch {
	case len(listRecordsResponse.Errors) > 0:
		return "", false, fmt.Errorf("%w: %s",
			errors.ErrUnsuccessfulResponse, strings.Join(listRecordsResponse.Errors, ","))
	case !listRecordsResponse.Success:
		return "", false, fmt.Errorf("%w", errors.ErrUnsuccessfulResponse)
	case len(listRecordsResponse.Result) == 0:
		return "", false, fmt.Errorf("%w", errors.ErrNoResultReceived)
	case len(listRecordsResponse.Result) > 1:
		return "", false, fmt.Errorf("%w: %d instead of 1",
			errors.ErrNumberOfResultsReceived, len(listRecordsResponse.Result))
	case listRecordsResponse.Result[0].Content == newIP.String(): // up to date
		return "", true, nil
	}
	return listRecordsResponse.Result[0].ID, false, nil
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}
	identifier, upToDate, err := p.getRecordID(ctx, client, ip)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrGetRecordID, err)
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
	err = encoder.Encode(requestData)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrRequestEncode, err)
	}

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
	err = decoder.Decode(&parsedJSON)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", errors.ErrUnmarshalResponse, err)
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
