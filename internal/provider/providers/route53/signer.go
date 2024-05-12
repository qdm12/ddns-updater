package route53

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

const route53Domain = "route53.amazonaws.com"
const v4SignatureVersion = "aws4_request"
const route53Service = "route53"

// Global resources needs signature to us-east-1 globalRegion
// and update / insert operations to route53 are also in us-east-1
const globalRegion = "us-east-1"

type credentials struct {
	accessKey string
	secretkey string
}

// Implements the signature v4 header based to upsert route53 domains
// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
type v4Signer struct {
	credentials credentials
	scope       scope
}

type scope struct {
	region           string
	service          string
	signatureVersion string
}

func (v4 *v4Signer) Sign(req *http.Request, payload []byte, date time.Time) (string, error) {
	if err := v4.sanatizeHostHeader(req); err != nil {
		return "", err
	}

	sanatizedHeaders, err := v4.sanatizeHeaders(req.Header)
	if err != nil {
		return "", err
	}

	canonicalRequest, signedHeaders := v4.buildCanonicalRequest(req.Method, req.URL.Path, sanatizedHeaders, payload)
	credentialScope := v4.formatScope(date)
	stringToSign := v4.buildStringToSign(canonicalRequest, credentialScope, date)
	signingKey := v4.buildPrivKey(date)
	signature := hmacSha256Sum([]byte(signingKey), []byte(stringToSign))

	credential := fmt.Sprintf("%s/%s", v4.credentials.accessKey, credentialScope)
	return fmt.Sprintf("AWS4-HMAC-SHA256 Credential=%s,SignedHeaders=%s,Signature=%s", credential, signedHeaders, hex.EncodeToString(signature)), nil
}

func (v4 *v4Signer) buildCanonicalRequest(method, path string, headers map[string]string, payload []byte) (string, string) {
	toSignHeaders := make([]string, 0, len(headers))
	for header := range headers {
		toSignHeaders = append(toSignHeaders, header)
	}
	sort.Strings(toSignHeaders)
	signedHeaders := strings.Join(toSignHeaders, ";")

	formatedHeaders := make([]string, 0, len(headers))
	for _, header := range toSignHeaders {
		formatedHeaders = append(formatedHeaders, fmt.Sprintf("%s:%s", header, headers[header]))
	}
	canonicalHeaders := strings.Join(formatedHeaders, "\n") + "\n"

	canonicalMethod := strings.ToUpper(method)
	canonicalPath := pathEncoder(path)
	canonicalQuery := "" // no query arg used
	bodyHash := sha256Sum(payload)
	return strings.Join([]string{
		canonicalMethod,
		canonicalPath,
		canonicalQuery,
		canonicalHeaders,
		signedHeaders,
		hex.EncodeToString(bodyHash),
	}, "\n"), signedHeaders
}

func (v4 *v4Signer) buildStringToSign(canonicalRequest, credentialScope string, date time.Time) string {
	hashRequest := sha256Sum([]byte(canonicalRequest))
	dateTime := formatDateTime(date)
	return strings.Join([]string{
		"AWS4-HMAC-SHA256",
		dateTime,
		credentialScope,
		hex.EncodeToString(hashRequest),
	}, "\n")
}

func (v4 *v4Signer) buildPrivKey(date time.Time) string {
	dateOnly := formateDate(date)
	signingKey := []byte("AWS4" + v4.credentials.secretkey)
	for _, value := range [][]byte{
		[]byte(dateOnly),
		[]byte(v4.scope.region),
		[]byte(v4.scope.service),
		[]byte(v4.scope.signatureVersion),
	} {
		signingKey = hmacSha256Sum(signingKey, value)
	}
	return string(signingKey)
}

func (v4 *v4Signer) sanatizeHeaders(headers http.Header) (map[string]string, error) {
	// These are the only mandatory headers as no other x-amz header is expected for now
	mandatoryHeaders := map[string]bool{
		"content-type": true,
		"host":         true,
	}

	sanitizedHeaders := map[string]string{}

	for header := range headers {
		lowerCasedHeader := strings.ToLower(header)

		if _, ok := mandatoryHeaders[lowerCasedHeader]; !ok {
			continue
		}

		headerValues := make([]string, 0, len(headers[header]))
		for _, value := range headers[header] {
			headerValues = append(headerValues, strings.Trim(value, " \t\n\v\f\r"))
		}

		// headers with different cases can happen in a request e.g. Content-type and Content-Type
		if values, ok := sanitizedHeaders[lowerCasedHeader]; ok {
			sanitizedHeaders[lowerCasedHeader] = fmt.Sprintf("%s,%s", values, strings.Join(headerValues, ","))
		} else {
			sanitizedHeaders[lowerCasedHeader] = strings.Join(headerValues, ",")
		}
	}

	missingHeaders := []string{}
	for header := range mandatoryHeaders {
		if _, ok := sanitizedHeaders[header]; !ok {
			missingHeaders = append(missingHeaders, header)
		}
	}

	if len(missingHeaders) > 0 {
		return map[string]string{}, fmt.Errorf("missing mandatory header(s) to sign request: '%s'", strings.Join(missingHeaders, "', '"))
	}
	return sanitizedHeaders, nil
}

func (v4 *v4Signer) sanatizeHostHeader(req *http.Request) error {
	// Remove any existing host header in order to enforce route53.amazonaws.com
	for header := range req.Header {
		if strings.ToLower(header) == "host" {
			req.Header.Del(header)
		}
	}

	domain := req.URL.Host
	if req.Host != "" {
		domain = req.Host // authoritaive pseudo-header is preferred
	}

	if domain != route53Domain {
		return fmt.Errorf("request must be to %s: %s", route53Domain, req.Host)
	}

	req.Header.Set("Host", route53Domain)
	return nil
}

func (v4 *v4Signer) formatScope(date time.Time) string {
	dateOnly := formateDate(date)
	return fmt.Sprintf("%s/%s/%s/%s", dateOnly, v4.scope.region, v4.scope.service, v4.scope.signatureVersion)
}

func formatDateTime(date time.Time) string {
	// Amazon only accept this format, golang native RFC3339 doesn't work
	return date.Format("20060102T150405Z")
}

func formateDate(date time.Time) string {
	// Amazon only accept this format, golang native DateOnly doesn't work
	return date.Format("20060102")
}

func pathEncoder(path string) string {
	// Amazon specific replacements, see their uriEncode guideline
	// https://docs.aws.amazon.com/AmazonS3/latest/API/sig-v4-header-based-auth.html
	replacer := strings.NewReplacer("%2F", "/", "+", "%20")
	res := url.PathEscape(path)
	return replacer.Replace(res)
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
