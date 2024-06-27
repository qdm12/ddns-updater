package porkbun

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

// See https://porkbun.com/api/json/v3/documentation#DNS%20Retrieve%20Records%20by%20Domain,%20Subdomain%20and%20Type
func (p *Provider) getRecordIDs(ctx context.Context, client *http.Client, recordType string) (
	recordIDs []string, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/retrieveByNameType/" + p.domain + "/" + recordType + "/",
	}
	if p.owner != "@" && p.owner != "*" {
		u.Path += p.owner
	}

	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return nil, fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return nil, fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, makeErrorMessage(response.Body))
	}

	var responseData struct {
		Records []struct {
			ID string `json:"id"`
		} `json:"records"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&responseData)
	if err != nil {
		return nil, fmt.Errorf("json decoding response body: %w", err)
	}

	for _, record := range responseData.Records {
		recordIDs = append(recordIDs, record.ID)
	}

	return recordIDs, nil
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Create%20Record
func (p *Provider) createRecord(ctx context.Context, client *http.Client,
	recordType string, ipStr string) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/create/" + p.domain,
	}
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
		Name:         p.owner,
		TTL:          strconv.FormatUint(uint64(p.ttl), 10),
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, makeErrorMessage(response.Body))
	}
	return nil
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Edit%20Record%20by%20Domain%20and%20ID
func (p *Provider) updateRecord(ctx context.Context, client *http.Client,
	recordType string, ipStr string, recordID string) (err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/edit/" + p.domain + "/" + recordID,
	}
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
		Name:         p.owner,
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, makeErrorMessage(response.Body))
	}
	return nil
}

// See https://porkbun.com/api/json/v3/documentation#DNS%20Delete%20Records%20by%20Domain,%20Subdomain%20and%20Type
func (p *Provider) deleteAliasRecord(ctx context.Context, client *http.Client) (err error) {
	var subdomain string
	if p.owner != "@" {
		subdomain = p.owner + "."
	}
	u := url.URL{
		Scheme: "https",
		Host:   "porkbun.com",
		Path:   "/api/json/v3/dns/deleteByNameType/" + p.domain + "/ALIAS/" + subdomain,
	}
	postRecordsParams := struct {
		SecretAPIKey string `json:"secretapikey"`
		APIKey       string `json:"apikey"`
	}{
		SecretAPIKey: p.secretAPIKey,
		APIKey:       p.apiKey,
	}
	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(postRecordsParams)
	if err != nil {
		return fmt.Errorf("json encoding request data: %w", err)
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), buffer)
	if err != nil {
		return fmt.Errorf("creating http request: %w", err)
	}
	setHeaders(request)

	response, err := client.Do(request)
	if err != nil {
		return fmt.Errorf("doing http request: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, makeErrorMessage(response.Body))
	}
	return nil
}
