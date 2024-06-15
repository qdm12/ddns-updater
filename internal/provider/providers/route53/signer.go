package route53

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"
)

const (
	route53Domain  = "route53.amazonaws.com"
	dateTimeFormat = "20060102T150405Z"
	dateFormat     = "20060102"
)

// signer implements the signature v4 header based to upsert route53 domains
// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
type signer struct {
	accessKey        string
	secretkey        string
	region           string
	service          string
	signatureVersion string
}

func (s *signer) sign(method, urlPath string, payload []byte, date time.Time) (
	headerValue string) {
	credentialScope := fmt.Sprintf("%s/%s/%s/%s", date.Format(dateFormat),
		s.region, s.service, s.signatureVersion)
	credential := fmt.Sprintf("%s/%s", s.accessKey, credentialScope)
	const signedHeaders = "content-type;host"
	canonicalRequest := buildCanonicalRequest(method, urlPath, signedHeaders, payload)
	stringToSign := buildStringToSign(date, canonicalRequest, credentialScope)
	signingKey := s.buildPrivateKey(date)
	signature := hmacSha256Sum([]byte(signingKey), []byte(stringToSign))
	signatureString := hex.EncodeToString(signature)
	return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s,SignedHeaders=%s,Signature=%s",
		credential, signedHeaders, signatureString)
}

func buildCanonicalRequest(method, path, headers string, payload []byte) (
	canonicalRequest string) {
	canonicalHeaders := "content-type:application/xml\nhost:" + route53Domain + "\n"
	const canonicalQuery = "" // no query arg used
	payloadHashDigest := hex.EncodeToString(sha256Sum(payload))
	canonicalRequest = strings.Join([]string{
		strings.ToUpper(method),
		path,
		canonicalQuery,
		canonicalHeaders,
		headers,
		payloadHashDigest,
	}, "\n")
	return canonicalRequest
}

func buildStringToSign(date time.Time,
	canonicalRequest, credentialScope string) string {
	return "AWS4-HMAC-SHA256\n" +
		date.Format(dateTimeFormat) + "\n" +
		credentialScope + "\n" +
		hex.EncodeToString(sha256Sum([]byte(canonicalRequest)))
}

func (s *signer) buildPrivateKey(date time.Time) string {
	signingKey := []byte("AWS4" + s.secretkey)
	for _, value := range [][]byte{
		[]byte(date.Format(dateFormat)),
		[]byte(s.region),
		[]byte(s.service),
		[]byte(s.signatureVersion),
	} {
		signingKey = hmacSha256Sum(signingKey, value)
	}
	return string(signingKey)
}

func sha256Sum(d []byte) []byte {
	hasher := sha256.New()
	hasher.Write(d)
	return hasher.Sum(nil)
}

func hmacSha256Sum(key, value []byte) []byte {
	hasher := hmac.New(sha256.New, key)
	hasher.Write(value)
	return hasher.Sum(nil)
}
