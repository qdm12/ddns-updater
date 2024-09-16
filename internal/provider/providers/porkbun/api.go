package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

type dnsRecord struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Content  string `json:"content"`
	TTL      string `json:"ttl"`
	Priority string `json:"prio"`
	Notes    string `json:"notes"`
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Retrieve%20Records%20by%20Domain,%20Subdomain%20and%20Type
func (p *Provider) getRecords(ctx context.Context, client *http.Client, recordType, owner string) (
	records []dnsRecord, err error) {
	url := "https://porkbun.com/api/json/v3/dns/retrieveByNameType/" + p.domain + "/" + recordType + "/"
	if owner != "@" {
		// Note Porkbun requires we send the unescaped '*' character.
		url += owner
	}

	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
	}

	type jsonResponseData struct {
		Records []dnsRecord `json:"records"`
	}
	const decodeBody = true
	responseData, err := httpPost[jsonResponseData](ctx, client, url, postRecordsParams, decodeBody)
	if err != nil {
		return nil, fmt.Errorf("for record type %s and record owner %s: %w",
			recordType, owner, err)
	}

	return responseData.Records, nil
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Create%20Record
func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType, owner, ipStr string) (err error) {
	url := "https://porkbun.com/api/json/v3/dns/create/" + p.domain
	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
		Content      string `json:"content"`
		Name         string `json:"name,omitempty"`
		Type         string `json:"type"`
		TTL          string `json:"ttl"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
		Content:      ipStr,
		Type:         recordType,
		Name:         owner,
		TTL:          strconv.FormatUint(uint64(p.ttl), 10),
	}
	const decodeBody = false
	_, err = httpPost[struct{}](ctx, client, url, postRecordsParams, decodeBody)
	if err != nil {
		return fmt.Errorf("for record type %s and record owner %s: %w",
			recordType, owner, err)
	}

	return nil
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Edit%20Record%20by%20Domain%20and%20ID
func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordType, owner, ipStr, recordID string) (err error) {
	url := "https://porkbun.com/api/json/v3/dns/edit/" + p.domain + "/" + recordID
	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
		Content      string `json:"content"`
		Type         string `json:"type"`
		TTL          string `json:"ttl"`
		Name         string `json:"name,omitempty"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
		Content:      ipStr,
		Type:         recordType,
		TTL:          strconv.FormatUint(uint64(p.ttl), 10),
		Name:         owner,
	}
	const decodeBody = false
	_, err = httpPost[struct{}](ctx, client, url, postRecordsParams, decodeBody)
	if err != nil {
		return fmt.Errorf("for record type %s, record owner %s and record id %s: %w",
			recordType, owner, recordID, err)
	}

	return nil
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Delete%20Records%20by%20Domain,%20Subdomain%20and%20Type
func (p *Provider) deleteRecord(ctx context.Context, client *http.Client, recordType, owner string) (err error) {
	url := "https://porkbun.com/api/json/v3/dns/deleteByNameType/" + p.domain + "/" + recordType + "/"
	if owner != "@" {
		// Note Porkbun requires we send the unescaped '*' character.
		url += owner
	}
	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
	}

	const decodeBody = false
	_, err = httpPost[struct{}](ctx, client, url, postRecordsParams, decodeBody)
	if err != nil {
		return fmt.Errorf("for record type %s and record owner %s: %w",
			recordType, owner, err)
	}

	return nil
}

func httpPost[T any](ctx context.Context, client *http.Client, //nolint:ireturn
	url string, requestData any, decodeBody bool) (responseData T, err error) {
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(requestData)
	if err != nil {
		return responseData, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, url, buffer)
	if err != nil {
		return responseData, fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return responseData, fmt.Errorf("doing http request: %w", err)
	}

	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return responseData, fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, makeErrorMessage(response.Body))
	}

	if decodeBody {
		decoder := json.NewDecoder(response.Body)
		err = decoder.Decode(&responseData)
		if err != nil {
			_ = response.Body.Close()
			return responseData, fmt.Errorf("json decoding response body: %w", err)
		}
	}

	err = response.Body.Close()
	if err != nil {
		return responseData, fmt.Errorf("closing response body: %w", err)
	}

	return responseData, nil
}
