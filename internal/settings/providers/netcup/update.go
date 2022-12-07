package netcup

import (
	"fmt"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"golang.org/x/net/context"
)

func (p *Provider) getRecordToUpdate(ctx context.Context,
	client *http.Client, session string, ip netip.Addr) (
	record dnsRecord, err error) {
	recordSet, err := p.infoDNSRecords(ctx, client, session)
	if err != nil {
		return record, err
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	found := false
	for _, record = range recordSet.DNSRecords {
		if record.Hostname == p.host && record.Type == recordType {
			found = true
			break
		}
	}

	if found {
		record.Destination = ip.String()
	} else {
		record = dnsRecord{
			Hostname:    p.host,
			Type:        recordType,
			Destination: ip.String(),
		}
	}

	return record, nil
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
