package settings

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/network"
	netlib "github.com/qdm12/golibs/network"
	"github.com/qdm12/golibs/verification"
)

//nolint:maligned
type cloudflare struct {
	domain         string
	host           string
	ipVersion      models.IPVersion
	dnsLookup      bool
	key            string
	token          string
	email          string
	userServiceKey string
	zoneIdentifier string
	identifier     string
	proxied        bool
	ttl            uint
}

func NewCloudflare(data json.RawMessage, domain, host string, ipVersion models.IPVersion, noDNSLookup bool) (s Settings, err error) {
	extraSettings := struct {
		Key            string `json:"key"`
		Token          string `json:"token"`
		Email          string `json:"email"`
		UserServiceKey string `json:"user_service_key"`
		ZoneIdentifier string `json:"zone_identifier"`
		Identifier     string `json:"identifier"`
		Proxied        bool   `json:"proxied"`
		TTL            uint   `json:"ttl"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	c := &cloudflare{
		domain:         domain,
		host:           host,
		ipVersion:      ipVersion,
		dnsLookup:      !noDNSLookup,
		key:            extraSettings.Key,
		token:          extraSettings.Token,
		email:          extraSettings.Email,
		userServiceKey: extraSettings.UserServiceKey,
		zoneIdentifier: extraSettings.ZoneIdentifier,
		identifier:     extraSettings.Identifier,
		proxied:        extraSettings.Proxied,
		ttl:            extraSettings.TTL,
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
		case !constants.MatchCloudflareKey(c.key):
			return fmt.Errorf("invalid key format")
		case !verification.NewVerifier().MatchEmail(c.email):
			return fmt.Errorf("invalid email format")
		}
	case len(c.userServiceKey) > 0: // only user service key
		if !constants.MatchCloudflareKey(c.key) {
			return fmt.Errorf("invalid user service key format")
		}
	default: // API token only
		if !constants.MatchCloudflareToken(c.token) {
			return fmt.Errorf("invalid API token key format")
		}
	}
	switch {
	case len(c.zoneIdentifier) == 0:
		return fmt.Errorf("zone identifier cannot be empty")
	case len(c.identifier) == 0:
		return fmt.Errorf("identifier cannot be empty")
	case c.ttl == 0:
		return fmt.Errorf("TTL cannot be left to 0")
	}
	return nil
}

func (c *cloudflare) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: Cloudflare]", c.domain, c.host)
}

func (c *cloudflare) Domain() string {
	return c.domain
}

func (c *cloudflare) Host() string {
	return c.host
}

func (c *cloudflare) IPVersion() models.IPVersion {
	return c.ipVersion
}

func (c *cloudflare) DNSLookup() bool {
	return c.dnsLookup
}

func (c *cloudflare) BuildDomainName() string {
	return buildDomainName(c.host, c.domain)
}

func (c *cloudflare) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", c.BuildDomainName(), c.BuildDomainName())),
		Host:      models.HTML(c.Host()),
		Provider:  "<a href=\"https://www.cloudflare.com\">Cloudflare</a>",
		IPVersion: models.HTML(c.ipVersion),
	}
}

func (c *cloudflare) Update(client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	type cloudflarePutBody struct {
		Type    string `json:"type"`    // forced to A
		Name    string `json:"name"`    // DNS record name i.e. example.com
		Content string `json:"content"` // ip address
		Proxied bool   `json:"proxied"` // whether the record is receiving the performance and security benefits of Cloudflare
		TTL     uint   `json:"ttl"`
	}
	u := url.URL{
		Scheme: "https",
		Host:   "api.cloudflare.com",
		Path:   fmt.Sprintf("/client/v4/zones/%s/dns_records/%s", c.zoneIdentifier, c.identifier),
	}
	r, err := network.BuildHTTPPut(
		u.String(),
		cloudflarePutBody{
			Type:    "A",
			Name:    c.host,
			Content: ip.String(),
			Proxied: c.proxied,
			TTL:     c.ttl,
		},
	)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	switch {
	case len(c.token) > 0:
		r.Header.Set("Authorization", "Bearer "+c.token)
	case len(c.userServiceKey) > 0:
		r.Header.Set("X-Auth-User-Service-Key", c.userServiceKey)
	case len(c.email) > 0 && len(c.key) > 0:
		r.Header.Set("X-Auth-Email", c.email)
		r.Header.Set("X-Auth-Key", c.key)
	default:
		return nil, fmt.Errorf("email and key are both unset and user service key is not set and no token was provided")
	}
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return nil, err
	} else if status > http.StatusUnsupportedMediaType {
		return nil, fmt.Errorf("HTTP status %d", status)
	}
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
	if err := json.Unmarshal(content, &parsedJSON); err != nil {
		return nil, err
	} else if !parsedJSON.Success {
		var errStr string
		for _, e := range parsedJSON.Errors {
			errStr += fmt.Sprintf("error %d: %s; ", e.Code, e.Message)
		}
		return nil, fmt.Errorf(errStr)
	}
	newIP = net.ParseIP(parsedJSON.Result.Content)
	if newIP == nil {
		return nil, fmt.Errorf("new IP %q is malformed", parsedJSON.Result.Content)
	} else if !newIP.Equal(ip) {
		return nil, fmt.Errorf("new IP address %s is not %s", newIP.String(), ip.String())
	}
	return newIP, nil
}
