package godaddy

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
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	key       string
	secret    string
	matcher   regex.Matcher
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (p *provider, err error) {
	extraSettings := struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
		secret:    extraSettings.Secret,
		matcher:   matcher,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case !p.matcher.GodaddyKey(p.key):
		return errors.ErrMalformedKey
	case len(p.secret) == 0:
		return errors.ErrEmptySecret
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString(p.domain, p.host, constants.GoDaddy, p.ipVersion)
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
		Provider:  "<a href=\"https://godaddy.com\">GoDaddy</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAuthSSOKey(request, p.key, p.secret)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}
	type goDaddyPutBody struct {
		Data string `json:"data"` // IP address to update to
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.godaddy.com",
		Path:   fmt.Sprintf("/v1/domains/%s/records/%s/%s", p.domain, recordType, p.host),
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	requestData := []goDaddyPutBody{
		{Data: ip.String()},
	}
	if err := encoder.Encode(requestData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrRequestEncode, err)
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

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	err = fmt.Errorf("%w: %d", errors.ErrBadHTTPStatus, response.StatusCode)
	var parsedJSON struct {
		Message string `json:"message"`
	}
	jsonErr := json.Unmarshal(b, &parsedJSON)
	if jsonErr != nil || len(parsedJSON.Message) == 0 {
		return nil, fmt.Errorf("%w: %s", err, utils.ToSingleLine(string(b)))
	}
	return nil, fmt.Errorf("%w: %s", err, parsedJSON.Message)
}
