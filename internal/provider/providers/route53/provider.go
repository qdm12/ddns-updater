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
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain     string
	host       string
	ipVersion  ipversion.IPVersion
	ipv6Suffix netip.Prefix
	zoneID     string
	ttl        uint32
	signer     *signer
}

type settings struct {
	AccessKey string  `json:"access_key"`
	SecretKey string  `json:"secret_key"`
	ZoneID    string  `json:"zone_id"`
	TTL       *uint32 `json:"ttl,omitempty"`
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	provider *Provider, err error) {
	var providerSpecificSettings settings
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("decoding provider specific settings: %w", err)
	}

	err = validateSettings(providerSpecificSettings, domain, host)
	if err != nil {
		return nil, fmt.Errorf("validating provider specific settings: %w", err)
	}

	const defaultTTL = 300
	ttl := uint32(defaultTTL)
	if providerSpecificSettings.TTL != nil {
		ttl = *providerSpecificSettings.TTL
	}

	// Global resources needs signature to us-east-1 globalRegion
	// and update / insert operations to route53 are also in us-east-1.
	const globalRegion = "us-east-1"
	const route53Service = "route53"
	const v4SignatureVersion = "aws4_request"
	signer := &signer{
		accessKey:        providerSpecificSettings.AccessKey,
		secretkey:        providerSpecificSettings.SecretKey,
		region:           globalRegion,
		service:          route53Service,
		signatureVersion: v4SignatureVersion,
	}

	return &Provider{
		domain:     domain,
		host:       host,
		ipVersion:  ipVersion,
		ipv6Suffix: ipv6Suffix,
		signer:     signer,
		zoneID:     providerSpecificSettings.ZoneID,
		ttl:        ttl,
	}, nil
}

func validateSettings(providerSpecificSettings settings, domain, host string) error {
	switch {
	case domain == "":
		return fmt.Errorf("%w", errors.ErrDomainNotSet)
	case host == "":
		return fmt.Errorf("%w", errors.ErrHostNotSet)
	case providerSpecificSettings.AccessKey == "":
		return fmt.Errorf("%w", errors.ErrAccessKeyNotSet)
	case providerSpecificSettings.SecretKey == "":
		return fmt.Errorf("%w", errors.ErrSecretKeyNotSet)
	case providerSpecificSettings.ZoneID == "":
		return fmt.Errorf("%w", errors.ErrZoneIdentifierNotSet)
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

// See https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   route53Domain,
		Path:   "/2013-04-01/hostedzone/" + p.zoneID + "/rrset",
	}

	changeRRSetRequest := newChangeRRSetRequest(p.BuildDomainName(), p.ttl, ip)

	// Note the AWS API does not accept JSON for this endpoint
	buffer := bytes.NewBuffer(nil)
	encoder := xml.NewEncoder(buffer)
	err = encoder.Encode(changeRRSetRequest)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("XML encoding change RRSet request: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	p.setHeaders(request, buffer.Bytes())

	response, err := client.Do(request)
	if err != nil {
		return netip.Addr{}, err
	}
	defer response.Body.Close()

	xmlDecoder := xml.NewDecoder(response.Body)
	if response.StatusCode == http.StatusOK {
		return ip, nil
	}

	var errorResponse errorResponse
	err = xmlDecoder.Decode(&errorResponse)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("XML decoding response body: %w", err)
	}
	return netip.Addr{}, fmt.Errorf("%w: %d: request %s %s/%s: %s",
		errors.ErrHTTPStatusNotValid, response.StatusCode,
		errorResponse.RequestID, errorResponse.Error.Type,
		errorResponse.Error.Code, errorResponse.Error.Message)
}

func (p *Provider) setHeaders(request *http.Request, payload []byte) {
	now := time.Now().UTC()
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/xml")
	headers.SetAccept(request, "application/xml")
	request.Header.Set("Date", now.Format(dateTimeFormat))
	request.Header.Set("Host", route53Domain)
	signature := p.signer.sign(request.Method, request.URL.Path, payload, now)
	request.Header.Set("Authorization", signature)
}
