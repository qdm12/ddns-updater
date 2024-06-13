package route53

import (
	"encoding/xml"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
)

const recordAction = "UPSERT"
const xmlns = "https://route53.amazonaws.com/doc/2013-04-01/"

// Refer to https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html#API_ChangeResourceRecordSets_RequestSyntax
type changeResourceRecordSetsRequest struct {
	XMLName     xml.Name `xml:"ChangeResourceRecordSetsRequest"`
	ChangeBatch changeBatch
	XMLNS       string `xml:"xmlns,attr"`
}

// Refer to https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html#API_ChangeResourceRecordSets_ResponseSyntax
type changeResourceRecordSetsResponse struct {
	XMLNS      string   `xml:"xmlns,attr"`
	XMLName    xml.Name `xml:"ChangeResourceRecordSetsResponse"`
	ChangeInfo struct {
		Id          string
		Status      string
		SubmittedAt string
	}
}

// Refer to https://docs.aws.amazon.com/Route53/latest/APIReference/requests-rest-responses.html
type errorResponse struct {
	XMLNS   string   `xml:"xmlns,attr"`
	XMLName xml.Name `xml:"ErrorResponse"`
	Error   struct {
		Type    string
		Code    string
		Message string
	}
	RequestId string
}

type changeBatch struct {
	Changes []change `xml:"Changes>Change"`
}

type change struct {
	Action            string
	ResourceRecordSet resourceRecordSet
}

type resourceRecordSet struct {
	Name            string
	Type            string
	TTL             int32
	ResourceRecords []resourceRecord `xml:"ResourceRecords>ResourceRecord"`
}

type resourceRecord struct {
	Value string
}

func (p *Provider) simpleRecordChange(ip netip.Addr) changeResourceRecordSetsRequest {
	recordType := constants.A
	if p.ipVersion == ipversion.IP6 {
		recordType = constants.AAAA
	}

	return changeResourceRecordSetsRequest{
		XMLNS: xmlns,
		ChangeBatch: changeBatch{
			Changes: []change{
				{
					Action: recordAction,
					ResourceRecordSet: resourceRecordSet{
						Name: p.BuildDomainName(),
						Type: recordType,
						TTL:  p.ttl,
						ResourceRecords: []resourceRecord{
							{
								Value: ip.String(),
							},
						},
					},
				},
			},
		},
	}
}

func (p *Provider) setHeaders(req *http.Request, payload []byte) error {
	now := time.Now().UTC()
	headers.SetUserAgent(req)
	headers.SetContentType(req, "application/xml")
	headers.SetAccept(req, "application/xml")
	req.Header.Set("Date", formatDateTime(now))
	signature, err := p.signer.Sign(req, payload, now)
	if err != nil {
		return fmt.Errorf("signing request: %w", err)
	}
	req.Header.Set("Authorization", signature)
	return nil
}
