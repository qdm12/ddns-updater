package settings

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
	"github.com/qdm12/ddns-updater/internal/regex"
)

type ovh struct {
	domain        string
	host          string
	ipVersion     models.IPVersion
	username      string
	password      string
	useProviderIP bool
	mode          string
	apiEndpoint   string
	appKey        string
	appSecret     string
	consumerKey   string
}

func NewOVH(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
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
	o := &ovh{
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
	if err := o.isValid(); err != nil {
		return nil, err
	}
	return o, nil
}

func (o *ovh) isValid() error {
	if o.mode == "api" {
		switch {
		case len(o.appKey) == 0:
			return ErrEmptyAppKey
		case len(o.consumerKey) == 0:
			return ErrEmptyConsumerKey
		case len(o.appSecret) == 0:
			return ErrEmptySecret
		}
	} else {
		switch {
		case len(o.username) == 0:
			return ErrEmptyUsername
		case len(o.password) == 0:
			return ErrEmptyPassword
		case o.host == "*":
			return ErrHostWildcard
		}
	}
	return nil
}

func (o *ovh) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: OVH]", o.domain, o.host)
}

func (o *ovh) Domain() string {
	return o.domain
}

func (o *ovh) Host() string {
	return o.host
}

func (o *ovh) IPVersion() models.IPVersion {
	return o.ipVersion
}

func (o *ovh) Proxied() bool {
	return false
}

func (o *ovh) BuildDomainName() string {
	return buildDomainName(o.host, o.domain)
}

func (o *ovh) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", o.BuildDomainName(), o.BuildDomainName())),
		Host:      models.HTML(o.Host()),
		Provider:  "<a href=\"https://www.ovh.com/\">OVH DNS</a>",
		IPVersion: models.HTML(o.ipVersion),
	}
}

func (o *ovh) updateWithDynHost(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	u := url.URL{
		Scheme: "https",
		User:   url.UserPassword(o.username, o.password),
		Host:   "www.ovh.com",
		Path:   "/nic/update",
	}
	values := url.Values{}
	values.Set("system", "dyndns")
	values.Set("hostname", o.BuildDomainName())
	if !o.useProviderIP {
		values.Set("myip", ip.String())
	}
	u.RawQuery = values.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	setUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	b, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrUnmarshalResponse, err)
	}
	s := string(b)

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", ErrBadHTTPStatus, response.StatusCode, s)
	}

	switch {
	case strings.HasPrefix(s, notfqdn):
		return nil, ErrHostnameNotExists
	case strings.HasPrefix(s, "badrequest"):
		return nil, ErrBadRequest
	case strings.HasPrefix(s, "good"):
		return ip, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownResponse, s)
	}
}

func (o *ovh) updateWithZoneDNS(ctx context.Context, client *ovhClient.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	var ipStr string
	if ip.To4() == nil { // IPv6
		recordType = AAAA
		ipStr = ip.To16().String()
	} else {
		ipStr = ip.To4().String()
	}
	// subDomain filter of the ovh api expect an empty string to get @ record
	subDomain := o.host
	if subDomain == "@" {
		subDomain = ""
	}
	// get existing records
	var recordIDs []uint64
	url := fmt.Sprintf("/domain/zone/%s/record?fieldType=%s&subDomain=%s", o.domain, recordType, subDomain)
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
		url := fmt.Sprintf("/domain/zone/%s/record", o.domain)
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
			url := fmt.Sprintf("/domain/zone/%s/record/%d", o.domain, recordID)
			if err := client.PutWithContext(ctx, url, &putRecordsParams, nil); err != nil {
				return nil, err
			}
		}
	}

	url = fmt.Sprintf("/domain/zone/%s/refresh", o.domain)
	if err := client.PostWithContext(ctx, url, nil, nil); err != nil {
		return nil, err
	}

	return ip, nil
}

func (o *ovh) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	if o.mode != "api" {
		return o.updateWithDynHost(ctx, client, ip)
	}
	const defaultEndpoint = "ovh-eu"
	apiEndpoint := defaultEndpoint
	if len(o.apiEndpoint) > 0 {
		apiEndpoint = o.apiEndpoint
	}
	ovhClientInstance, err := ovhClient.NewClient(
		apiEndpoint,
		o.appKey,
		o.appSecret,
		o.consumerKey,
	)
	if err != nil {
		return nil, err
	}
	return o.updateWithZoneDNS(ctx, ovhClientInstance, ip)
}
