package porkbun

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain       string
	host         string
	ttl          uint
	ipVersion    ipversion.IPVersion
	apiKey       string
	secretAPIKey string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *Provider, err error) {
	extraSettings := struct {
		SecretAPIKey string `json:"secret_api_key"`
		APIKey       string `json:"api_key"`
		TTL          uint   `json:"ttl"`
	}{}
	err = json.Unmarshal(data, &extraSettings)
	if err != nil {
		return nil, err
	}
	p = &Provider{
		domain:       domain,
		host:         host,
		ipVersion:    ipVersion,
		secretAPIKey: extraSettings.SecretAPIKey,
		apiKey:       extraSettings.APIKey,
		ttl:          extraSettings.TTL,
	}
	err = p.isValid()
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (p *Provider) isValid() error {
	switch {
	case p.apiKey == "":
		return fmt.Errorf("%w", errors.ErrAPIKeyNotSet)
	case p.secretAPIKey == "":
		return fmt.Errorf("%w", errors.ErrAPISecretNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.host, constants.Porkbun, p.ipVersion)
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
	return false
}

func (p *Provider) BuildDomainName() string {
	return utils.BuildDomainName(p.host, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Host:      p.Host(),
		Provider:  "<a href=\"https://www.porkbun.com/\">Porkbun DNS</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func setHeaders(request *http.Request) {
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
}

// See https://porkbun.com/api/json/v3/documentation
func (p *Provider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	ipStr := ip.String()
	recordIDs, err := p.getRecordIDs(ctx, client, recordType)
	if err != nil {
		return netip.Addr{}, err
	}
	if len(recordIDs) == 0 {
		err = p.createRecord(ctx, client, recordType, ipStr)
		if err != nil {
			return netip.Addr{}, err
		}
		return ip, nil
	}

	for _, recordID := range recordIDs {
		err = p.updateRecord(ctx, client, recordType, ipStr, recordID)
		if err != nil {
			return netip.Addr{}, err
		}
	}

	return ip, nil
}
