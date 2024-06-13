package route53

import (
	"encoding/xml"
	"net/netip"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// It is easy to break XML parsing due to some missing label or top level
// node. This should be updated that changes to data structure does not break
// parsing.
func Test_changeResourceRecordSetsRequest_XML_Encode(t *testing.T) {
	t.Parallel()

	request := changeResourceRecordSetsRequest{
		XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
		ChangeBatch: changeBatch{
			Changes: []change{{
				Action: "UPSERT",
				ResourceRecordSet: resourceRecordSet{
					Name:            "test.com",
					Type:            "A",
					TTL:             uint32(300),
					ResourceRecords: []resourceRecord{{Value: "127.0.0.1"}},
				},
			}},
		},
	}
	body, err := xml.Marshal(request)
	require.NoError(t, err)
	const expectedBody = `<ChangeResourceRecordSetsRequest` +
		` xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` +
		`<ChangeBatch><Changes><Change><Action>UPSERT</Action>` +
		`<ResourceRecordSet><Name>test.com</Name><Type>A</Type><TTL>300</TTL>` +
		`<ResourceRecords><ResourceRecord><Value>127.0.0.1</Value></ResourceRecord>` +
		`</ResourceRecords></ResourceRecordSet></Change></Changes>` +
		`</ChangeBatch></ChangeResourceRecordSetsRequest>`
	assert.Equal(t, expectedBody, string(body))
}

func Test_changeResourceRecordSetsResponse_XML_Decode(t *testing.T) {
	t.Parallel()

	const response = `<?xml version="1.0"?>` +
		`<ChangeResourceRecordSetsResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` +
		`<ChangeInfo><Id>/change/FFFFFFFFFFFFFFFFFFFFF</Id><Status>PENDING</Status>` +
		`<SubmittedAt>2024-11-19T15:00:00.000Z</SubmittedAt>` +
		`</ChangeInfo></ChangeResourceRecordSetsResponse>
`

	var parsed changeResourceRecordSetsResponse
	err := xml.Unmarshal([]byte(response), &parsed)
	require.NoError(t, err)
	expectedObject := changeResourceRecordSetsResponse{
		XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
		XMLName: xml.Name{
			Space: "https://route53.amazonaws.com/doc/2013-04-01/",
			Local: "ChangeResourceRecordSetsResponse"},
		ChangeInfo: xmlChangeInfo{
			ID:          "/change/FFFFFFFFFFFFFFFFFFFFF",
			Status:      "PENDING",
			SubmittedAt: "2024-11-19T15:00:00.000Z",
		},
	}
	assert.Equal(t, expectedObject, parsed)
}

func Test_errorResponse_XML_Decode(t *testing.T) {
	t.Parallel()

	const response = `<?xml version="1.0"?><ErrorResponse xmlns="https://route53.amazonaws.com/doc/2013-04-01/">` +
		`<Error><Type>Sender</Type><Code>SignatureDoesNotMatch</Code><Message>` +
		`Signature not yet current: 20240518T160100Z is still later than 20240518T140900Z (20240518T140400Z + 5 min.)` +
		`</Message></Error><RequestId>ffffffff-ffff-ffff-ffff-ffffffffffff</RequestId></ErrorResponse>
`
	var parsed errorResponse
	err := xml.Unmarshal([]byte(response), &parsed)
	require.NoError(t, err)
	expectedObject := errorResponse{
		XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
		XMLName: xml.Name{
			Space: "https://route53.amazonaws.com/doc/2013-04-01/",
			Local: "ErrorResponse"},
		Error: xmlError{
			Type: "Sender",
			Code: "SignatureDoesNotMatch",
			Message: "Signature not yet current: " +
				"20240518T160100Z is still later than 20240518T140900Z" +
				" (20240518T140400Z + 5 min.)",
		},
		RequestID: "ffffffff-ffff-ffff-ffff-ffffffffffff",
	}
	assert.Equal(t, expectedObject, parsed)
}

func Test_newChangeRRSetRequest(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		name     string
		ttl      uint32
		ip       netip.Addr
		expected changeResourceRecordSetsRequest
	}{
		"ipv4": {
			name: "test.com",
			ttl:  300,
			ip:   netip.MustParseAddr("127.0.0.1"),
			expected: changeResourceRecordSetsRequest{
				XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
				ChangeBatch: changeBatch{
					Changes: []change{{
						Action: "UPSERT",
						ResourceRecordSet: resourceRecordSet{
							Name:            "test.com",
							Type:            "A",
							TTL:             300,
							ResourceRecords: []resourceRecord{{Value: "127.0.0.1"}},
						},
					}},
				},
			},
		},
		"ipv6": {
			name: "test.com",
			ttl:  300,
			ip:   netip.MustParseAddr("::1"),
			expected: changeResourceRecordSetsRequest{
				XMLNS: "https://route53.amazonaws.com/doc/2013-04-01/",
				ChangeBatch: changeBatch{
					Changes: []change{{
						Action: "UPSERT",
						ResourceRecordSet: resourceRecordSet{
							Name:            "test.com",
							Type:            "AAAA",
							TTL:             300,
							ResourceRecords: []resourceRecord{{Value: "::1"}},
						},
					}},
				},
			},
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			actual := newChangeRRSetRequest(testCase.name, testCase.ttl, testCase.ip)
			assert.Equal(t, testCase.expected, actual)
		})
	}
}
