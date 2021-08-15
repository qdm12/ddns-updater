package aliyun

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/utils"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

type provider struct {
	domain          string
	host            string
	ipVersion       ipversion.IPVersion
	regionId        string
	accessKeyId     string
	accessKeySecret string
}

func New(data json.RawMessage, domain, host string,
	ipVersion ipversion.IPVersion) (p *provider, err error) {
	extraSettings := struct {
		RegionId        string `json:"region_id"`
		AccessKeyId     string `json:"access_key_id"`
		AccessKeySecret string `json:"access_key_secret"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	p = &provider{
		domain:          domain,
		host:            host,
		ipVersion:       ipVersion,
		regionId:        extraSettings.RegionId,
		accessKeyId:     extraSettings.AccessKeyId,
		accessKeySecret: extraSettings.AccessKeySecret,
	}
	if err := p.isValid(); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *provider) isValid() error {
	switch {
	case len(p.regionId) == 0:
		return errors.ErrEmptyRegionId
	case len(p.accessKeyId) == 0:
		return errors.ErrEmptyAccessKeyId
	case len(p.accessKeySecret) == 0:
		return errors.ErrEmptyAccessKeySecret
	}
	return nil
}

func (p *provider) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: aliyun]", p.domain, p.host)
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

func (p *provider) newClient() *sdk.Client {
	client, err := sdk.NewClientWithAccessKey(p.regionId, p.accessKeyId, p.accessKeySecret)
	if err != nil {
		// Handle exceptions
		panic(err)
	}
	return client
}

func (p *provider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", p.BuildDomainName(), p.BuildDomainName())),
		Host:      models.HTML(p.Host()),
		Provider:  "<a href=\"https://aliyun.com/\">AliDNS</a>",
		IPVersion: models.HTML(p.ipVersion.String()),
	}
}

func (d *provider) getRecord(recordType string, client *sdk.Client) (record alidns.Record, err error) {
	request := alidns.CreateDescribeSubDomainRecordsRequest()
	request.SubDomain = d.BuildDomainName()
	request.Type = recordType

	response := alidns.CreateDescribeSubDomainRecordsResponse()
	err = client.DoAction(request, response)
	if err != nil {
		return alidns.Record{}, err
	}

	if len(response.DomainRecords.Record) == 0 {
		return alidns.Record{}, errors.ErrRecordNotFound
	} else if response.DomainRecords.Record[0].RecordId == "" {
		return alidns.Record{}, errors.ErrRecordIDNotFound
	}

	return response.DomainRecords.Record[0], nil
}

func (p *provider) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := constants.A
	var ipStr string
	if ip.To4() == nil { // IPv6
		recordType = constants.AAAA
	}
	ipStr = ip.String()
	aliClient := p.newClient()
	record, err := p.getRecord(recordType, aliClient)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", errors.ErrGetRecordID, err)
	}

	if record.Value == ipStr {
		return ip, nil
	}

	request := alidns.CreateUpdateDomainRecordRequest()
	request.RR = p.host
	request.RecordId = record.RecordId
	request.Type = recordType
	request.Value = ipStr

	response := alidns.CreateUpdateDomainRecordResponse()

	err = aliClient.DoAction(request, response)
	if err != nil {
		return nil, err
	}

	return ip, nil
}
