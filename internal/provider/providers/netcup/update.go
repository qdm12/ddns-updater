package netcup

import (
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"golang.org/x/net/context"
)

func (p *Provider) getRecordToUpdate(ctx context.Context,
	client *http.Client, session string, ip netip.Addr) (
	record dnsRecord, err error) {
	recordSet, err := p.infoDNSRecords(ctx, client, session)
	if err != nil {
		return record, fmt.Errorf("getting DNS records: %w", err)
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	for _, record = range recordSet.DNSRecords {
		if record.Hostname == p.host && record.Type == recordType {
			record.Destination = ip.String()
			return record, nil
		}
	}

	return dnsRecord{
		Hostname:    p.host,
		Type:        recordType,
		Destination: ip.String(),
	}, nil
}

func (p *Provider) updateDNSRecords(ctx context.Context, client *http.Client,
	session string, recordSet dnsRecordSet) (response dnsRecordSet, err error) {
	type jsonParam struct {
		APIKey         string       `json:"apikey"`
		APISessionID   string       `json:"apisessionid"`
		CustomerNumber string       `json:"customernumber"`
		DomainName     string       `json:"domainname"`
		DNSRecordSet   dnsRecordSet `json:"dnsrecordset"`
	}
	type jsonRequest struct {
		Action string    `json:"action"`
		Param  jsonParam `json:"param"`
	}

	request := jsonRequest{
		Action: "updateDnsRecords",
		Param: jsonParam{
			APIKey:         p.apiKey,
			APISessionID:   session,
			CustomerNumber: p.customerNumber,
			DomainName:     p.domain,
			DNSRecordSet:   recordSet,
		},
	}

	err = doJSONHTTP(ctx, client, request, &response)
	if err != nil {
		return response, fmt.Errorf("doing JSON HTTP exchange: %w", err)
	}

	return response, nil
}
