package route53

import (
	"encoding/xml"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
)

// See https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html#API_ChangeResourceRecordSets_RequestSyntax
type changeResourceRecordSetsRequest struct {
	XMLName     xml.Name    `xml:"ChangeResourceRecordSetsRequest"`
	ChangeBatch changeBatch `xml:"ChangeBatch"`
	XMLNS       string      `xml:"xmlns,attr"`
}

type changeBatch struct {
	Changes []change `xml:"Changes>Change"`
}

type change struct {
	Action            string            `xml:"Action"`
	ResourceRecordSet resourceRecordSet `xml:"ResourceRecordSet"`
}

type resourceRecordSet struct {
	Name            string           `xml:"Name"`
	Type            string           `xml:"Type"`
	TTL             uint32           `xml:"TTL"`
	ResourceRecords []resourceRecord `xml:"ResourceRecords>ResourceRecord"`
}

type resourceRecord struct {
	Value string `xml:"Value"`
}

// See https://docs.aws.amazon.com/Route53/latest/APIReference/API_ChangeResourceRecordSets.html#API_ChangeResourceRecordSets_ResponseSyntax
type changeResourceRecordSetsResponse struct {
	XMLNS      string        `xml:"xmlns,attr"`
	XMLName    xml.Name      `xml:"ChangeResourceRecordSetsResponse"`
	ChangeInfo xmlChangeInfo `xml:"ChangeInfo"`
}

type xmlChangeInfo struct {
	ID          string `xml:"Id"`
	Status      string `xml:"Status"`
	SubmittedAt string `xml:"SubmittedAt"`
}

// See https://docs.aws.amazon.com/Route53/latest/APIReference/requests-rest-responses.html
type errorResponse struct {
	XMLNS     string   `xml:"xmlns,attr"`
	XMLName   xml.Name `xml:"ErrorResponse"`
	Error     xmlError `xml:"Error"`
	RequestID string   `xml:"RequestId"`
}

type xmlError struct {
	Type    string `xml:"Type"`
	Code    string `xml:"Code"`
	Message string `xml:"Message"`
}

func newChangeRRSetRequest(name string, ttl uint32, ip netip.Addr) changeResourceRecordSetsRequest {
	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}

	return changeResourceRecordSetsRequest{
		XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
		ChangeBatch: changeBatch{
			Changes: []change{{
				Action: "UPSERT",
				ResourceRecordSet: resourceRecordSet{
					Name: name,
					Type: recordType,
					TTL:  ttl,
					ResourceRecords: []resourceRecord{{
						Value: ip.String(),
					}}},
			}},
		},
	}
}
