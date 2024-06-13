package route53

import (
	"encoding/xml"
	"testing"

	"github.com/stretchr/testify/assert"
)

// It is easy to break XML parsing due to some missing label or top level
// node. This should be updated that changes to data structure does not break
// parsing.
func TestEncodeRequest(t *testing.T) {
	t.Parallel()

	simpleRecordSet := changeResourceRecordSetsRequest{
		XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
		ChangeBatch: changeBatch{
			Changes: []change{
				{
					Action: "UPSERT",
					ResourceRecordSet: resourceRecordSet{
						Name:            "test.com",
						Type:            "A",
						TTL:             uint32(300),
						ResourceRecords: []resourceRecord{{Value: "127.0.0.1"}},
					},
				},
			},
		},
	}
	body, err := xml.Marshal(simpleRecordSet)
	if assert.NoError(t, err) {
		const expectedBody = `<ChangeResourceRecordSetsRequest` +
			` xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` +
			`<ChangeBatch><Changes><Change><Action>UPSERT</Action>` +
			`<ResourceRecordSet><Name>test.com</Name><Type>A</Type><TTL>300</TTL>` +
			`<ResourceRecords><ResourceRecord><Value>127.0.0.1</Value></ResourceRecord>` +
			`</ResourceRecords></ResourceRecordSet></Change></Changes>` +
			`</ChangeBatch></ChangeResourceRecordSetsRequest>`
		assert.EqualValues(t, expectedBody, string(body))
	}
}

func TestDecodeSuccessfullResponse(t *testing.T) {
	t.Parallel()

	const response = `<?xml version="1.0"?>` +
		`<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` +
		`<ChangeInfo><Id>/change/FFFFFFFFFFFFFFFFFFFFF</Id><Status>PENDING</Status>` +
		`<SubmittedAt>2024-11-19T15:00:00.000Z</SubmittedAt>` +
		`</ChangeInfo></ChangeResourceRecordSetsResponse>
`

	var parsed changeResourceRecordSetsResponse
	err := xml.Unmarshal([]byte(response), &parsed)
	if assert.NoError(t, err) {
		expectedObject := changeResourceRecordSetsResponse{
			XMLNS:   "https://route53.amazonaws.com/doc/2013-04-01/",
			XMLName: xml.Name{Space: "https://route53.amazonaws.com/doc/2013-04-01/", Local: "ChangeResourceRecordSetsResponse"},
			ChangeInfo: xmlChangeInfo{
				ID:          "/change/FFFFFFFFFFFFFFFFFFFFF",
				Status:      "PENDING",
				SubmittedAt: "2024-11-19T15:00:00.000Z",
			},
		}
		assert.EqualValues(t, expectedObject, parsed)
	}
}

func TestDecodeFailureResponse(t *testing.T) {
	t.Parallel()

	const response = `<?xml version="1.0"?><ErrorResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` +
		`<Error><Type>Sender</Type><Code>SignatureDoesNotMatch</Code><Message>` +
		`Signature not yet current: 20240518T160100Z is still later than 20240518T140900Z (20240518T140400Z + 5 min.)` +
		`</Message></Error><RequestId>ffffffff-ffff-ffff-ffff-ffffffffffff</RequestId></ErrorResponse>
`

	var parsed errorResponse
	err := xml.Unmarshal([]byte(response), &parsed)
	if assert.NoError(t, err) {
		expectedObject := errorResponse{
			XMLNS:   "https://route53.amazonaws.com/doc/2013-04-01/",
			XMLName: xml.Name{Space: "https://route53.amazonaws.com/doc/2013-04-01/", Local: "ErrorResponse"},
			Error: xmlError{
				Type: "Sender",
				Code: "SignatureDoesNotMatch",
				Message: "Signature not yet current: " +
					"20240518T160100Z is still later than 20240518T140900Z" +
					" (20240518T140400Z + 5 min.)",
			},
			RequestID: "ffffffff-ffff-ffff-ffff-ffffffffffff",
		}
		assert.EqualValues(t, expectedObject, parsed)
	}
}
