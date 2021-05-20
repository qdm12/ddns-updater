package settings

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/alidns"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/regex"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"net"
	"net/http"
)

type aliyun struct {
	domain          string
	host            string
	ipVersion       ipversion.IPVersion
	regionId        string
	accessKeyId     string
	accessKeySecret string
}

func NewAliyun(data json.RawMessage, domain, host string, ipVersion ipversion.IPVersion,
	_ regex.Matcher) (s Settings, err error) {
	extraSettings := struct {
		RegionId        string `json:"region_id"`
		AccessKeyId     string `json:"access_key_id"`
		AccessKeySecret string `json:"access_key_secret"`
	}{}
	if err := json.Unmarshal(data, &extraSettings); err != nil {
		return nil, err
	}
	d := &aliyun{
		domain:          domain,
		host:            host,
		ipVersion:       ipVersion,
		regionId:        extraSettings.RegionId,
		accessKeyId:     extraSettings.AccessKeyId,
		accessKeySecret: extraSettings.AccessKeySecret,
	}
	if err := d.isValid(); err != nil {
		return nil, err
	}
	return d, nil
}

func (d *aliyun) isValid() error {
	switch {
	case len(d.regionId) == 0:
		return ErrEmptyRegionId
	case len(d.accessKeyId) == 0:
		return ErrEmptyAccessKeyId
	case len(d.accessKeySecret) == 0:
		return ErrEmptyAccessKeySecret
	}
	return nil
}

func (d *aliyun) String() string {
	return fmt.Sprintf("[domain: %s | host: %s | provider: aliyun]", d.domain, d.host)
}

func (d *aliyun) Domain() string {
	return d.domain
}

func (d *aliyun) Host() string {
	return d.host
}

func (d *aliyun) IPVersion() ipversion.IPVersion {
	return d.ipVersion
}

func (d *aliyun) Proxied() bool {
	return false
}

func (d *aliyun) BuildDomainName() string {
	return buildDomainName(d.host, d.domain)
}

func (d *aliyun) newClient() *sdk.Client {
	client, err := sdk.NewClientWithAccessKey(d.regionId, d.accessKeyId, d.accessKeySecret)
	if err != nil {
		// Handle exceptions
		panic(err)
	}
	return client
}

func (d *aliyun) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:    models.HTML(fmt.Sprintf("<a href=\"http://%s\">%s</a>", d.BuildDomainName(), d.BuildDomainName())),
		Host:      models.HTML(d.Host()),
		Provider:  "<a href=\"https://aliyun.com/\">AliDNS</a>",
		IPVersion: models.HTML(d.ipVersion.String()),
	}
}

func (d *aliyun) getRecord(recordType string, client *sdk.Client) (record alidns.Record, err error) {
	request := alidns.CreateDescribeSubDomainRecordsRequest()
	request.SubDomain = d.BuildDomainName()
	request.Type = recordType

	response := alidns.CreateDescribeSubDomainRecordsResponse()
	err = client.DoAction(request, response)
	if err != nil {
		return alidns.Record{}, err
	}

	if len(response.DomainRecords.Record) == 0 {
		return alidns.Record{}, ErrRecordNotFound
	} else if response.DomainRecords.Record[0].RecordId == "" {
		return alidns.Record{}, ErrRecordIDNotFound
	}

	return response.DomainRecords.Record[0], nil
}

func (d *aliyun) Update(ctx context.Context, client *http.Client, ip net.IP) (newIP net.IP, err error) {
	recordType := A
	var ipStr string
	if ip.To4() == nil { // IPv6
		recordType = AAAA
		ipStr = ip.To16().String()
	} else {
		ipStr = ip.To4().String()
	}

	aliClient := d.newClient()
	record, err := d.getRecord(recordType, aliClient)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", ErrGetRecordID, err)
	}

	if record.Value == ipStr {
		return ip, nil
	}

	request := alidns.CreateUpdateDomainRecordRequest()
	request.RR = d.host
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
