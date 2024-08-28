package azure

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/netip"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/dns/armdns"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider/constants"
	ddnserrors "github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type Provider struct {
	domain            string // aka zoneName
	owner             string // aka relativeRecordSetName
	ipVersion         ipversion.IPVersion
	ipv6Suffix        netip.Prefix
	tenantID          string
	clientID          string
	clientSecret      string
	subscriptionID    string
	resourceGroupName string
}

func New(data json.RawMessage, domain, owner string,
	ipVersion ipversion.IPVersion, ipv6Suffix netip.Prefix) (
	p *Provider, err error) {
	var providerSpecificSettings struct {
		TenantID          string `json:"tenant_id"`
		ClientID          string `json:"client_id"`
		ClientSecret      string `json:"client_secret"`
		SubscriptionID    string `json:"subscription_id"`
		ResourceGroupName string `json:"resource_group_name"`
	}
	err = json.Unmarshal(data, &providerSpecificSettings)
	if err != nil {
		return nil, fmt.Errorf("json decoding provider specific settings: %w", err)
	}
	err = validateSettings(domain, owner, providerSpecificSettings.TenantID,
		providerSpecificSettings.ClientID, providerSpecificSettings.ClientSecret,
		providerSpecificSettings.SubscriptionID, providerSpecificSettings.ResourceGroupName)
	if err != nil {
		return nil, fmt.Errorf("validating settings: %w", err)
	}
	return &Provider{
		domain:            domain,
		owner:             owner,
		ipVersion:         ipVersion,
		ipv6Suffix:        ipv6Suffix,
		tenantID:          providerSpecificSettings.TenantID,
		clientID:          providerSpecificSettings.ClientID,
		clientSecret:      providerSpecificSettings.ClientSecret,
		subscriptionID:    providerSpecificSettings.SubscriptionID,
		resourceGroupName: providerSpecificSettings.ResourceGroupName,
	}, nil
}

func validateSettings(domain, owner, tenantID, clientID,
	clientSecret, subscriptionID, resourceGroupName string) error {
	switch {
	case domain == "":
		return fmt.Errorf("%w", ddnserrors.ErrDomainNotSet)
	case owner == "":
		return fmt.Errorf("%w", ddnserrors.ErrOwnerNotSet)
	case tenantID == "":
		return fmt.Errorf("%w: tenant id", ddnserrors.ErrCredentialsNotSet)
	case clientID == "":
		return fmt.Errorf("%w: client id", ddnserrors.ErrCredentialsNotSet)
	case clientSecret == "":
		return fmt.Errorf("%w: client secret", ddnserrors.ErrCredentialsNotSet)
	case subscriptionID == "":
		return fmt.Errorf("%w: subscription id", ddnserrors.ErrKeyNotSet)
	case resourceGroupName == "":
		return fmt.Errorf("%w: resource group name", ddnserrors.ErrKeyNotSet)
	}
	return nil
}

func (p *Provider) String() string {
	return utils.ToString(p.domain, p.owner, constants.Azure, p.ipVersion)
}

func (p *Provider) Domain() string {
	return p.domain
}

func (p *Provider) Owner() string {
	return p.owner
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
	return utils.BuildDomainName(p.owner, p.domain)
}

func (p *Provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName()),
		Owner:     p.Owner(),
		Provider:  "<a href=\"https://azure.microsoft.com/en-us/services/dns/\">Azure</a>",
		IPVersion: p.ipVersion.String(),
	}
}

func ptrTo[T any](v T) *T { return &v }

func (p *Provider) Update(ctx context.Context, _ *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	var recordType armdns.RecordType
	if ip.Is4() {
		recordType = armdns.RecordTypeA
	} else {
		recordType = armdns.RecordTypeAAAA
	}

	client, err := p.createClient()
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating client: %w", err)
	}

	response, err := p.getRecordSet(ctx, client, recordType)
	if err != nil {
		azureErr := &azcore.ResponseError{}
		if errors.As(err, &azureErr) && azureErr.StatusCode == http.StatusNotFound {
			err = p.createRecordSet(ctx, client, ip)
			if err != nil {
				return netip.Addr{}, fmt.Errorf("creating record set: %w", err)
			}
		}
		return netip.Addr{}, fmt.Errorf("getting record set: %w", err)
	}

	err = p.updateRecordSet(ctx, client, response, ip)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("updating record set: %w", err)
	}
	return ip, nil
}
