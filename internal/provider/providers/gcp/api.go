package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/netip"
	"net/url"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
)

type recordResourceSet struct {
	// Name is the fqdn.
	Name string `json:"name"`
	// Rrdatas, as defined in RFC 1035 (section 5) and RFC 1034 (section 3.6.1)
	Rrdatas []string `json:"rrdatas,omitempty"`
	// TTL is the number of seconds that this RRSet can be cached by resolvers.
	TTL uint32 `json:"ttl"`
	// Type is the identifier of a record type. For example A or AAAA.
	Type string `json:"type"`
}

func (p *Provider) getRRSet(ctx context.Context, client *http.Client, fqdn, recordType string) (
	rrSet *recordResourceSet, err error) {
	urlPath := fmt.Sprintf("/dns/v1/projects/%s/managedZones/%s/rrsets/%s/%s",
		p.project, p.zone, fqdn, recordType)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, makeAPIURL(urlPath), nil)
	if err != nil {
		return nil, err
	}
	headers.SetUserAgent(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	switch {
	case response.StatusCode == http.StatusNoContent:
		err = response.Body.Close()
		return nil, err
	case response.StatusCode == http.StatusNotFound:
		errMessage := decodeError(response.Body)
		return nil, fmt.Errorf("%w: %s", errors.ErrRecordResourceSetNotFound, errMessage)
	case response.StatusCode >= http.StatusOK &&
		response.StatusCode < http.StatusMultipleChoices:
		rrSet = new(recordResourceSet)
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&rrSet)
		if err != nil {
			return nil, fmt.Errorf("json decoding rrset: %w", err)
		}
		err = response.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("closing response body: %w", err)
		}
		return rrSet, nil
	default:
		errMessage := decodeError(response.Body)
		return nil, fmt.Errorf("%w: %s", errors.ErrHTTPStatusNotValid, errMessage)
	}
}

func (p *Provider) createRRSet(ctx context.Context, client *http.Client, fqdn, recordType string,
	ip netip.Addr) (err error) {
	urlPath := fmt.Sprintf("/dns/v1/projects/%s/managedZones/%s/rrsets", p.project, p.zone)
	body := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(body)
	rrSet := &recordResourceSet{
		Name:    fqdn,
		Rrdatas: []string{ip.String()},
		Type:    recordType,
	}
	err = encoder.Encode(rrSet)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, makeAPIURL(urlPath), body)
	if err != nil {
		return err
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode >= http.StatusOK &&
		response.StatusCode < http.StatusMultipleChoices {
		return response.Body.Close()
	}
	errMessage := decodeError(response.Body)
	return fmt.Errorf("%w: %s", errors.ErrHTTPStatusNotValid, errMessage)
}

func (p *Provider) patchRRSet(ctx context.Context, client *http.Client, fqdn, recordType string,
	ip netip.Addr) (err error) {
	urlPath := fmt.Sprintf("/dns/v1/projects/%s/managedZones/%s/rrsets/%s/%s",
		p.project, p.zone, fqdn, recordType)
	body := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(body)
	rrSet := &recordResourceSet{
		Name:    fqdn,
		Rrdatas: []string{ip.String()},
		Type:    recordType,
	}
	err = encoder.Encode(rrSet)
	if err != nil {
		return err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, makeAPIURL(urlPath), body)
	if err != nil {
		return err
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	if response.StatusCode >= http.StatusOK &&
		response.StatusCode < http.StatusMultipleChoices {
		return response.Body.Close()
	}
	errMessage := decodeError(response.Body)
	return fmt.Errorf("%w: %s", errors.ErrHTTPStatusNotValid, errMessage)
}

func makeAPIURL(path string) string {
	urlValues := make(url.Values)
	urlValues.Set("alt", "json")
	urlValues.Set("prettyPrint", "false")
	u := url.URL{
		Scheme:   "https",
		Host:     "dns.googleapis.com",
		Path:     path,
		RawQuery: urlValues.Encode(),
	}
	return u.String()
}
