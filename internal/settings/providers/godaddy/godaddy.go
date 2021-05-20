package godaddy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
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

type godaddy struct {
	domain    string
	host      string
	ipVersion ipversion.IPVersion
	key       string
	secret    string
	matcher   regex.Matcher
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	matcher regex.Matcher) (g *godaddy, err error) {
	extraSettings := struct {
		Key    string `json:"key"`
		Secret string `json:"secret"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	g = &godaddy{
		domain:    domain,
		host:      host,
		ipVersion: ipVersion,
		key:       extraSettings.Key,
		secret:    extraSettings.Secret,
		matcher:   matcher,
	}
	if err := g.isValid(); err != nil {
		return nil, err
	}
	return g, nil
}

func (g *godaddy) isValid() error {
	switch {
	case !g.matcher.GodaddyKey(g.key):
		return errors.ErrMalformedKey
	case len(g.secret) == 0:
		return errors.ErrEmptySecret
	}
	return nil
}

func (g *godaddy) String() string {
	return utils.ToString(g.domain, g.host, constants.GoDaddy, g.ipVersion)
}

func (g *godaddy) Domain() string {
	return g.domain
}

func (g *godaddy) Host() string {
	return g.host
}

func (g *godaddy) IPVersion() ipversion.IPVersion {
	return g.ipVersion
}

func (g *godaddy) Proxied() bool {
	return false
}

func (g *godaddy) BuildDomainName() string {
	return utils.BuildDomainName(g.host, g.domain)
}

func (g *godaddy) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", g.BuildDomainName(), g.BuildDomainName())),
		Host:      models.HTML(g.Host()),
		Provider:  "<a href=\"https://godaddy.com\">GoDaddy</a>",
		IPVersion: models.HTML(g.ipVersion.String()),
	}
}

func (g *godaddy) setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetAuthSSOKey(request, g.key, g.secret)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

func (g *godaddy) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
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
		Path:   fmt.Sprintf("/v1/domains/%s/records/%s/%s", g.domain, recordType, g.host),
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
	g.setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	if response.StatusCode == http.StatusOK {
		return ip, nil
	}

	b, err := ioutil.ReadAll(response.Body)
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
