package netcup

import (
	"context"
	"net/http"
)

func (p *Provider) infoDNSRecords(ctx context.Context, client *http.Client,
	session string) (recordSet dnsRecordSet, err error) {
	type jsonParams struct {
		APIKey         string `json:"apikey"`
		APISessionID   string `json:"apisessionid"`
		CustomerNumber string `json:"customernumber"`
		DomainName     string `json:"domainname"`
	}

	type jsonRequest struct {
		Action string     `json:"action"`
		Param  jsonParams `json:"param"`
	}

	request := jsonRequest{
		Action: "infoDnsRecords",
		Param: jsonParams{
			APIKey:         p.apiKey,
			APISessionID:   session,
			CustomerNumber: p.customerNumber,
			DomainName:     p.domain,
		},
	}

	err = doJSONHTTP(ctx, client, request, &recordSet)
	return recordSet, err
}
