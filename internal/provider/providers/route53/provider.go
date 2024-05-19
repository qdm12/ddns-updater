package route53

import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain      string
	host        string
	ipVersion   ipversion.IPVersion
	ipv6Suffix  netip.Prefix
	credentials credentials
	zoneID      string
	ttl         int32
	signer      signer
}

type settings struct {
	AccessKey string `json:"access_key"`
	SecretKey string `json:"secret_key"`
	ZoneID    string `json:"zone_id"`
	TTL       *int32 `json:"ttl"`
}

type signer interface {
	Sign(*http.Request, []byte, time.Time) (string, error)
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error) {

	var providerSpecificSettings settings
	if err := json.Unmarshal(data, &providerSpecificSettings); err != nil {
		return nil, fmt.Errorf("decoding provider specific settings: %w", err)
	}

	if err := validateSettings(providerSpecificSettings, domain, host); err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	ttl := int32(300)
	if providerSpecificSettings.TTL != nil {
		ttl = *providerSpecificSettings.TTL
	}

	return &Provider{
		domain:     domain,
		host:       host,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		signer: &v4Signer{
			credentials: credentials{
				accessKey: providerSpecificSettings.AccessKey,
				secretkey: providerSpecificSettings.SecretKey,
			},
			scope: scope{
				region:           globalRegion,
				service:          route53Service,
				signatureVersion: v4SignatureVersion,
			},
		},
		zoneID: providerSpecificSettings.ZoneID,
		ttl:    ttl,
	}, nil
}

func validateSettings(providerSpecificSettings settings, domain, host string) error {
	switch {
	case domain == "":
		return fmt.Errorf("%w", errors.ErrDomainNotSet)
	case host == "":
		return fmt.Errorf("%w", errors.ErrHostNotSet)
	case providerSpecificSettings.AccessKey == "":
		return fmt.Errorf("%w", errors.ErrKeyNotSet)
	case providerSpecificSettings.SecretKey == "":
		return fmt.Errorf("%w", errors.ErrAccessKeySecretNotSet)
	case providerSpecificSettings.ZoneID == "":
		return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
	case providerSpecificSettings.TTL != nil && *providerSpecificSettings.TTL < 0:
		return fmt.Errorf("%w", errors.ErrTTLTooLow)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Route53, p.ipVersion)
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

func (p *Provider) IPv6Suffix() netip.Prefix {
	return p.ipv6Suffix
}

func (p *Provider) Proxied() bool {
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://aws.amazon.com/route53/\">Amazon Route 53</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	// API details https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html

	u := url.URL{
		Scheme: "https",
		Host:   route53Domain,
		Path:   fmt.Sprintf("/2013-04-01/hostedzone/%s/rrset", p.zoneID),
	}

	changeBatch := p.simpleRecordChange(ip)

	// AWS api does not accept application/json as input for this endpoint
	payload, err := xml.Marshal(changeBatch)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("encoding http body: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(payload))
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	// Signature based auth request
	if err := p.setHeaders(request, payload); err != nil {
		return netip.Addr{}, fmt.Errorf("%w", err)
	}

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	xmlDecoder := xml.NewDecoder(response.Body)
	if response.StatusCode != http.StatusOK {
		var errorResponse errorResponse
		if err := xmlDecoder.Decode(&errorResponse); err != nil {
			return netip.Addr{}, fmt.Errorf("decoding body to xml: %w", err)
		}
		return netip.Addr{}, fmt.Errorf("%w: %d: request %s %s/%s: %s", errors.ErrHTTPStatusNotValid, response.StatusCode, errorResponse.RequestId, errorResponse.Error.Type, errorResponse.Error.Code, errorResponse.Error.Message)
	}

	return ip, nil
}
