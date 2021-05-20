package ovh

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"strings"

	ovhClient "github.com/ovh/go-ovh/ovh"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain        string
	host          string
	ipVersion     ipversion.IPVersion
	username      string
	password      string
	useProviderIP bool
	mode          string
	apiEndpoint   string
	appKey        string
	appSecret     string
	consumerKey   string
}

func New(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		Username      string `json:"username"`
		Password      string `json:"password"`
		UseProviderIP bool   `json:"provider_ip"`
		Mode          string `json:"mode"`
		APIEndpoint   string `json:"api_endpoint"`
		AppKey        string `json:"app_key"`
		AppSecret     string `json:"app_secret"`
		ConsumerKey   string `json:"consumer_key"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:        domain,
		host:          host,
		ipVersion:     ipVersion,
		username:      extraSettings.Username,
		password:      extraSettings.Password,
		useProviderIP: extraSettings.UseProviderIP,
		mode:          extraSettings.Mode,
		apiEndpoint:   extraSettings.APIEndpoint,
		appKey:        extraSettings.AppKey,
		appSecret:     extraSettings.AppSecret,
		consumerKey:   extraSettings.ConsumerKey,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	if p.mode == "api" {
		switch {
		case len(p.appKey) == 0:
			return errors.ErrEmptyAppKey
		case len(p.consumerKey) == 0:
			return errors.ErrEmptyConsumerKey
		case len(p.appSecret) == 0:
			return errors.ErrEmptySecret
		}
	} else {
		switch {
		case len(p.username) == 0:
			return errors.ErrEmptyUsername
		case len(p.password) == 0:
			return errors.ErrEmptyPassword
		case p.host == "*":
			return errors.ErrHostWildcard
		}
	}
	return nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: OVH]", p.domain, p.host)
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
		Provider:  "<a href=\"https://www.ovh.com/\">OVH DNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (p *provider) updateWithDynHost(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(p.username, p.password),
		Host:   "www.ovh.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("system", "dyndns")
	values.Set("hostname", p.BuildDomainName())
	if !p.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrBadHTTPStatus, response.StatusCode, s)
	}

	switch {
	case strings.HasPrefix(s, constants.Notfqdn):
		return nil, errors.ErrHostnameNotExists
	case strings.HasPrefix(s, "badrequest"):
		return nil, errors.ErrBadRequest
	case strings.HasPrefix(s, "good"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", errors.ErrUnknownResponse, s)
	}
}

func (p *provider) updateWithZoneDNS(ctx context.Context, client *ovhClient.Client, ip net.IP) (
	newIP net.IP, err error) {
	recordType := constants.A
	var ipStr string
	if ip.To4() == nil { // IPv6
		recordType = constants.AAAA
		ipStr = ip.To16().String()
	} else {
		ipStr = ip.To4().String()
	}
	// subDomain filter of the ovh api expect an empty string to get @ record
	subDomain := p.host
	if subDomain == "@" {
		subDomain = ""
	}
	// get existing records
	var recordIDs []uint64
	url := fmt.Sprintf("/domain/zone/%s/record?fieldType=%s&subDomain=%s", p.domain, recordType, subDomain)
	if err := client.GetWithContext(ctx, url, &recordIDs); err != nil {
		return nil, err
	}
	if len(recordIDs) == 0 {
		// create a new record
		postRecordsParams := struct {
			FieldType string `json:"fieldType"`
			SubDomain string `json:"subDomain"`
			Target    string `json:"target"`
		}{
			FieldType: recordType,
			SubDomain: subDomain,
			Target:    ipStr,
		}
		url := fmt.Sprintf("/domain/zone/%s/record", p.domain)
		if err := client.PostWithContext(ctx, url, &postRecordsParams, nil); err != nil {
			return nil, err
		}
	} else {
		// update existing record
		putRecordsParams := struct {
			Target string `json:"target"`
		}{
			Target: ipStr,
		}
		for _, recordID := range recordIDs {
			url := fmt.Sprintf("/domain/zone/%s/record/%d", p.domain, recordID)
			if err := client.PutWithContext(ctx, url, &putRecordsParams, nil); err != nil {
				return nil, err
			}
		}
	}

	url = fmt.Sprintf("/domain/zone/%s/refresh", p.domain)
	if err := client.PostWithContext(ctx, url, nil, nil); err != nil {
		return nil, err
	}

	return ip, nil
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	if p.mode != "api" {
		return p.updateWithDynHost(ctx, client, ip)
	}
	const defaultEndpoint = "ovh-eu"
	apiEndpoint := defaultEndpoint
	if len(p.apiEndpoint) > 0 {
		apiEndpoint = p.apiEndpoint
	}
	ovhClientInstance, err := ovhClient.NewClient(
		apiEndpoint,
		p.appKey,
		p.appSecret,
		p.consumerKey,
	)
	if err != nil {
		return nil, err
	}
	return p.updateWithZoneDNS(ctx, ovhClientInstance, ip)
}
