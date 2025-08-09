package route53

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/signer/v4"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

const (
	route53Domain  = "route53.amazonaws.com"
	dateTimeFormat = time.RFC1123
)

type Route53Signer struct {
	signer *v4.Signer
}

func NewRoute53Signer(creds *credentials.Credentials) *Route53Signer {
	return &Route53Signer{
		signer: v4.NewSigner(creds),
	}
}

func updateRecord(ctx context.Context, client *http.Client, signer *Route53Signer, zoneID, domainName string, ttl uint32, ip netip.Addr) (netip.Addr, error) {
	u := url.URL{
		Scheme: "https",
		Host:   route53Domain,
		Path:   "/2013-04-01/hostedzone/" + zoneID + "/rrset",
	}

	changeRRSetRequest := newChangeRRSetRequest(domainName, ttl, ip)
	buffer := new(bytes.Buffer)
	encoder := xml.NewEncoder(buffer)
	if err := encoder.Encode(changeRRSetRequest); err != nil {
		return netip.Addr{}, fmt.Errorf("XML encoding change RRSet request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return netip.Addr{}, fmt.Errorf("creating http request: %w", err)
	}

	req.Header.Set("Content-Type", "application/xml")
	req.Header.Set("Accept", "application/xml")
	_, err = signer.signer.Sign(req, bytes.NewReader(buffer.Bytes()), "route53", "us-east-1", time.Now().UTC())
	if err != nil {
		return netip.Addr{}, fmt.Errorf("signing request: %w", err)
	}

	resp, err := client.Do(req)
	if err != nil {
		return netip.Addr{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return ip, nil
	}

	var errResp errorResponse
	if err := xml.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return netip.Addr{}, fmt.Errorf("XML decoding response body: %w", err)
	}
	return netip.Addr{}, fmt.Errorf("%w: %d: request %s %s/%s: %s",
		errors.ErrHTTPStatusNotValid, resp.StatusCode,
		errResp.RequestID, errResp.Error.Type,
		errResp.Error.Code, errResp.Error.Message)
}
