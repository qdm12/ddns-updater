package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	netlib "github.com/qdm12/golibs/network"
)

//nolint:maligned
type azure struct {
	domain                string
	host                  string
	ipVersion             models.IPVersion
	dnsLookup             bool
	subscriptionID        string
	resourceGroupName     string
	zoneName              string
	relativeRecordSetName string
}

func NewAzure(data json.RawMessage, domain, host string, ipVersion models.IPVersion,
	noDNSLookup bool, matcher regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		SubscriptionID        string `json:"subscription_id"`
		ResourceGroupName     string `json:"resource_group_name"`
		ZoneName              string `json:"zone_name"`
		RelativeRecordSetName string `json:"relative_record_set_name"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	a := &azure{
		domain:                domain,
		host:                  host,
		ipVersion:             ipVersion,
		dnsLookup:             !noDNSLookup,
		subscriptionID:        extraSettings.SubscriptionID,
		resourceGroupName:     extraSettings.ResourceGroupName,
		zoneName:              extraSettings.ZoneName,
		relativeRecordSetName: extraSettings.RelativeRecordSetName,
	}
	if err := a.isValid(); err != nil {
		return nil, err
	}
	return a, nil
}

func (a *azure) isValid() error {
	switch {
	case len(a.subscriptionID) == 0:
		return fmt.Errorf("subscription ID is empty")
	case len(a.resourceGroupName) == 0:
		return fmt.Errorf("resource group name is empty")
	case len(a.zoneName) == 0:
		return fmt.Errorf("zone name is empty")
	case len(a.relativeRecordSetName) == 0:
		return fmt.Errorf("relative record set name is empty")
	}
	return nil
}

func (a *azure) String() string {
	return toString(a.domain, a.host, constants.GOOGLE, a.ipVersion)
}

func (a *azure) Domain() string {
	return a.domain
}

func (a *azure) Host() string {
	return a.host
}

func (a *azure) DNSLookup() bool {
	return a.dnsLookup
}

func (a *azure) IPVersion() models.IPVersion {
	return a.ipVersion
}

func (a *azure) BuildDomainName() string {
	return buildDomainName(a.host, a.domain)
}

func (a *azure) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", a.BuildDomainName(), a.BuildDomainName())),
		Host:      models.HTML(a.Host()),
		Provider:  "<a href=\"https://azure.microsoft.com/en-us/services/dns/\">Azure</a>",
		IPVersion: models.HTML(a.ipVersion),
	}
}

func (a *azure) Update(ctx context.Context, client netlib.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	if ip.To4() == nil {
		recordType = AAAA
	}

	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnsZones/%s/%s/%s", //nolint:lll
		a.subscriptionID, a.resourceGroupName, a.zoneName, recordType, a.relativeRecordSetName)
	values := url.Values{}
	values.Set("api-version", "2018-05-01")
	u := url.URL{
		Scheme:   "https",
		Host:     "management.azure.com",
		Path:     path,
		RawQuery: values.Encode(),
	}

	type (
		ARecord struct {
			IPv4Address string `json:"ipv4Address"`
		}
		AAAARecord struct {
			IPv6Address string `json:"ipv6Address"`
		}
	)
	type recordSet struct {
		Properties struct {
			ARecords    []ARecord    `json:"ARecords"`
			AAAARecords []AAAARecord `json:"AAAARecords"`
		} `json:"properties"`
	}
	requestBody := recordSet{}
	if recordType == A {
		requestBody.Properties.ARecords = append(
			requestBody.Properties.ARecords,
			ARecord{IPv4Address: ip.String()})
	} else {
		requestBody.Properties.AAAARecords = append(
			requestBody.Properties.AAAARecords,
			AAAARecord{IPv6Address: ip.String()})
	}
	requestBytes, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}
	requestBuffer := bytes.NewBuffer(requestBytes)

	r, err := http.NewRequestWithContext(ctx, http.MethodPatch, u.String(), requestBuffer)
	if err != nil {
		return nil, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentia.mcgaw@gmail.com")
	content, status, err := client.Do(r)
	if err != nil {
		return nil, err
	} else if status != http.StatusOK {
		type cloudErrorBody struct {
			Code    string           `json:"code"`
			Message string           `json:"message"`
			Target  string           `json:"target"`
			Details []cloudErrorBody `json:"details"`
		}
		var response struct {
			Error cloudErrorBody `json:"error"`
		}
		if err := json.Unmarshal(content, &response); err != nil {
			return nil, fmt.Errorf("%s: cannot decode error response: %w", http.StatusText(status), err)
		}
		return nil, fmt.Errorf("%s: %s (target: %s)", response.Error.Code, response.Error.Message, response.Error.Target)
	}
	var response recordSet
	if err := json.Unmarshal(content, &response); err != nil {
		return nil, fmt.Errorf("cannot decode success response: %w", err)
	}
	if recordType == A {
		if n := len(response.Properties.ARecords); n != 1 {
			return nil, fmt.Errorf("response contains %d A records instead of 1", n)
		}
		record := response.Properties.ARecords[0]
		newIP = net.ParseIP(record.IPv4Address)
		if newIP == nil {
			return nil, fmt.Errorf("IPv4 address in response is not valid: %s", record.IPv4Address)
		} else if newIP.To4() == nil {
			return nil, fmt.Errorf("IP address in response is not an IPv4 address: %s", record.IPv4Address)
		}
		return newIP, nil
	}
	// AAAA
	if n := len(response.Properties.AAAARecords); n != 1 {
		return nil, fmt.Errorf("response contains %d AAAA records instead of 1", n)
	}
	record := response.Properties.AAAARecords[0]
	newIP = net.ParseIP(record.IPv6Address)
	if newIP == nil {
		return nil, fmt.Errorf("IPv6 address in response is not valid: %s", record.IPv6Address)
	} else if newIP.To4() != nil {
		return nil, fmt.Errorf("IP address in response is not an IPv6 address: %s", record.IPv6Address)
	}
	return newIP, nil
}
