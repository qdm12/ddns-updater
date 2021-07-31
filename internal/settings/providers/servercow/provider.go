package servercow

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/log"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	username 			string
	host          string
	domain        string
	ipVersion     ipversion.IPVersion
	password      string
	useProviderIP bool
	ttl						uint
	logger        log.Logger
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion, logger log.Logger) (p *provider, err error) {
	extraSettings := struct {
		Username         string `json:"username"`
		Password         string `json:"password"`
		Domain					 string `json:"domain"`
		TTL							 uint		`json:"ttl"`
		UseProviderIP bool   `json:"provider_ip"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password: 		 extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
		domain:				 extraSettings.Domain,
		logger:        logger,
		ttl:					 extraSettings.TTL,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case p.username == "":
		return errors.ErrEmptyUsername
	case p.password == "":
		return errors.ErrEmptyPassword
	case p.host == "@":
		return errors.ErrHostAt
	}
	return nil
}

func (p *provider) String() string {
	return utils.ToString("servercow.de", p.host, constants.Servercow, p.ipVersion)
}

func (p *provider) Domain() string {
	return "servercow.de"
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
	return utils.BuildDomainName(p.host, "servercow.de")
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://servercow.de\">Servercow</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	if ip.To4() == nil {
		recordType = constants.AAAA
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.servercow.de",
		Path:   fmt.Sprintf("/dns/v1/domains/%s", p.domain),
	}
	values := url.Values{}
	u.RawQuery = values.Encode()
	if !p.useProviderIP {
		if ip.To4() == nil {
			values.Set("ip6", ip.String())
		} else {
			values.Set("ip", ip.String())
		}
	}

	requestData := struct {
		Type    string `json:"type"`    // constants.A or constants.AAAA depending on ip address given
		Name    string `json:"name"`    // DNS record name i.e. example.com
		Content string `json:"content"` // ip address
		TTL     uint   `json:"ttl"`
	}{
		Type:    recordType,
		Name:    p.host,
		Content: ip.String(),
		TTL:     p.ttl,
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	if err := encoder.Encode(requestData); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrRequestEncode, err)
	}

	p.logger.Debug("HTTP POST: " + u.String() + ": " + buffer.String())

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, err
	}
	headers.SetContentType(request, "application/json")
	headers.SetXAuthUsername(request, p.username)
	headers.SetXAuthPassword(request, p.password)
	headers.SetUserAgent(request)

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
		Message string `json:"message"`
		Error string `json:"error"`
	}

	if err := decoder.Decode(&parsedJSON); err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	if(parsedJSON.Message != "ok") {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, parsedJSON.Error)
	}

	return ip, nil
}
