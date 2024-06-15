package route53

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func Test_signer_sign(t *testing.T) {
	t.Parallel()

	signer := &signer{
		accessKey:        "AKIDEXAMPLE",
		secretkey:        "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		region:           "us-east-1",
		service:          "route53",
		signatureVersion: "aws4_request",
	}

	const method = http.MethodPost
	const urlPath = "/2013-04-01/hostedzone/Z148QEXAMPLE8V/rrset"
	payload := []byte{1, 2, 3, 4, 5}
	date := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	headerValue := signer.sign(method, urlPath, payload, date)

	const expected = "AWS4-HMAC-SHA256 Credential=AKIDEXAMPLE" +
		"/20210101/us-east-1/route53/aws4_request," +
		"SignedHeaders=content-type;host," +
		"Signature=441038f6dd576fcb8c6426efd92615b705d8bd14394809aca9daf059256c61be"
	assert.Equal(t, expected, headerValue)
}

func Test_buildCanonicalRequest(t *testing.T) {
	t.Parallel()

	const method = http.MethodPost
	const urlPath = "/2013-04-01/hostedzone/Z148QEXAMPLE8V/rrset"
	const headers = "content-type;host"
	payload := []byte{1, 2, 3, 4, 5}

	canonicalRequest := buildCanonicalRequest(method, urlPath, headers, payload)

	const expected = "POST\n/2013-04-01/hostedzone/Z148QEXAMPLE8V/rrset\n\n" +
		"content-type:application/xml\nhost:route53.amazonaws.com\n\n" +
		"content-type;host\n74f81fe167d99b4cb41d6d0ccda82278caee9f3e2f25d5e5a3936ff3dcec60d0"
	assert.Equal(t, expected, canonicalRequest)
}

func Test_buildStringToSign(t *testing.T) {
	t.Parallel()

	date := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	const canonicalRequest = "canonical request"
	const credentialScope = "20210101/us-east-1/route53/aws4_request"
	stringToSign := buildStringToSign(date, canonicalRequest, credentialScope)
	const expected = "AWS4-HMAC-SHA256\n20210101T000000Z\n" +
		"20210101/us-east-1/route53/aws4_request\n" +
		"6148e80fc369360885d29b93dbc72ca5f4107f6609fadc47a77b036ef241f2bf"
	assert.Equal(t, expected, stringToSign)
}

func Test_signer_buildPrivateKey(t *testing.T) {
	t.Parallel()

	date := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	signer := &signer{
		secretkey:        "wJalrXUtnFEMI/K7MDENG+bPxRfiCYEXAMPLEKEY",
		region:           "us-east-1",
		service:          "route53",
		signatureVersion: "aws4_request",
	}
	privateKey := signer.buildPrivateKey(date)
	const expectedPrivateKey = "\xa4\x13\x97\x935\x9d\x0f\xa6\xe6܉]\xfb\x83p\x85iJp\xe0\xb5\xf0͕E\xb5Jp\xe6,\x9e\xf7"
	assert.Equal(t, expectedPrivateKey, privateKey)
}
