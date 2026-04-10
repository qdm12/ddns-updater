package azure

import (
	"bytes"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider/constants"
	"github.com/qdm12/ddns-updater/internal/provider/errors"
	"github.com/qdm12/ddns-updater/internal/provider/headers"
	"github.com/qdm12/ddns-updater/internal/provider/utils"
)

type rrSet struct {
	ID         string `json:"id"`
	Etag       string `json:"etag"`
	Name       string `json:"name"`
	Type       string `json:"type"`
	Properties struct {
		Metadata    map[string]string `json:"metadata"`
		TTL         uint32            `json:"TTL"`
		FQDN        string            `json:"fqdn"`
		ARecords    []arecord         `json:"ARecords"`
		AAAARecords []aaaarecord      `json:"AAAARecords"`
	} `json:"properties"`
}

type arecord struct {
	IPv4Address string `json:"ipv4Address"`
}

type aaaarecord struct {
	IPv6Address string `json:"ipv6Address"`
}

var (
	errTokenTypeNotBearer = stderrors.New("token type is not bearer")
	errTokenEmpty         = stderrors.New("access token is empty")
	errTokenExpiresInZero = stderrors.New("access token expires_in is zero")
)

func (p *Provider) getAccessToken(ctx context.Context, client *http.Client) (err error) {
	requestBody := url.Values{
		"grant_type":    {"client_credentials"},
		"client_id":     {p.clientID},
		"client_secret": {p.clientSecret},
		"scope":         {"https://management.azure.com/.default"},
	}.Encode()

	tokenURL := fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", p.tenantID)
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, strings.NewReader(requestBody))
	if err != nil {
		return err
	}

	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/x-www-form-urlencoded")
	headers.SetAccept(request, "application/json")

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		message := decodeError(response.Body)
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, message)
	}

	var data struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   uint32 `json:"expires_in"`
		TokenType   string `json:"token_type"`
	}
	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	_ = response.Body.Close()
	switch {
	case err != nil:
		return fmt.Errorf("JSON decoding token response: %w", err)
	case strings.ToLower(data.TokenType) != "bearer":
		return fmt.Errorf("%w: %s", errTokenTypeNotBearer, data.TokenType)
	case data.AccessToken == "":
		return fmt.Errorf("%w", errTokenEmpty)
	case data.ExpiresIn == 0:
		return fmt.Errorf("%w", errTokenExpiresInZero)
	}

	const safetyMargin = 5 * time.Second
	p.tokenExp = p.now().Add(time.Duration(data.ExpiresIn)*time.Second - safetyMargin)
	p.token = data.AccessToken

	return nil
}

func makeURL(subscriptionID, resourceGroupName, domain, recordType, host string) string {
	path := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s/providers/Microsoft.Network/dnsZones/%s/%s/%s",
		subscriptionID, resourceGroupName, domain, recordType, host)
	values := url.Values{}
	values.Set("api-version", "2018-05-01")
	u := url.URL{
		Scheme:   "https",
		Host:     "management.azure.com",
		Path:     path,
		RawQuery: values.Encode(),
	}
	return u.String()
}

func (p *Provider) getRecordSet(ctx context.Context, client *http.Client,
	recordType string,
) (data rrSet, err error) {
	url := makeURL(p.subscriptionID, p.resourceGroupName, p.domain, recordType, p.owner)
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return data, err
	}
	headers.SetUserAgent(request)
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)

	response, err := client.Do(request)
	if err != nil {
		return data, err
	}

	switch response.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		_ = response.Body.Close()
		return data, fmt.Errorf("%w: %s %s",
			errors.ErrRecordNotFound, p.owner, recordType)
	default:
		message := decodeError(response.Body)
		_ = response.Body.Close()
		return data, fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, message)
	}

	decoder := json.NewDecoder(response.Body)
	err = decoder.Decode(&data)
	_ = response.Body.Close()
	if err != nil {
		return data, fmt.Errorf("JSON decoding response: %w", err)
	}

	return data, nil
}

func (p *Provider) createRecordSet(ctx context.Context, client *http.Client,
	ip netip.Addr,
) (err error) {
	var data rrSet
	recordType := constants.A
	if ip.Is4() {
		data.Properties.ARecords = []arecord{{IPv4Address: ip.String()}}
	} else {
		recordType = constants.AAAA
		data.Properties.AAAARecords = []aaaarecord{{IPv6Address: ip.String()}}
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("JSON encoding request body: %w", err)
	}

	url := makeURL(p.subscriptionID, p.resourceGroupName, p.domain, recordType, p.owner)

	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, buffer)
	if err != nil {
		return err
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		message := decodeError(response.Body)
		_ = response.Body.Close()
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, message)
	}

	err = response.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	return nil
}

func (p *Provider) updateRecordSet(ctx context.Context, client *http.Client,
	data rrSet, ip netip.Addr,
) (err error) {
	recordType := constants.A
	if ip.Is4() {
		if len(data.Properties.ARecords) == 0 {
			data.Properties.ARecords = make([]arecord, 1)
		}
		for i := range data.Properties.ARecords {
			data.Properties.ARecords[i].IPv4Address = ip.String()
		}
	} else {
		recordType = constants.AAAA
		if len(data.Properties.AAAARecords) == 0 {
			data.Properties.AAAARecords = make([]aaaarecord, 1)
		}
		for i := range data.Properties.AAAARecords {
			data.Properties.AAAARecords[i].IPv6Address = ip.String()
		}
	}

	buffer := bytes.NewBuffer(nil)
	encoder := json.NewEncoder(buffer)
	err = encoder.Encode(data)
	if err != nil {
		return fmt.Errorf("JSON encoding request body: %w", err)
	}
	url := makeURL(p.subscriptionID, p.resourceGroupName, p.domain, recordType, p.owner)
	request, err := http.NewRequestWithContext(ctx, http.MethodPatch, url, buffer)
	if err != nil {
		return err
	}
	headers.SetUserAgent(request)
	headers.SetContentType(request, "application/json")
	headers.SetAccept(request, "application/json")
	headers.SetAuthBearer(request, p.token)
	if data.Etag != "" {
		request.Header.Add("If-Match", data.Etag)
	}

	response, err := client.Do(request)
	if err != nil {
		return err
	}

	if response.StatusCode != http.StatusOK {
		message := decodeError(response.Body)
		_ = response.Body.Close()
		return fmt.Errorf("%w: %d: %s", errors.ErrHTTPStatusNotValid,
			response.StatusCode, message)
	}

	err = response.Body.Close()
	if err != nil {
		return fmt.Errorf("closing response body: %w", err)
	}

	return nil
}

func decodeError(body io.Reader) (message string) {
	type cloudErrorBody struct {
		Code    string           `json:"code"`
		Message string           `json:"message"`
		Target  string           `json:"target"`
		Details []cloudErrorBody `json:"details"`
	}
	var errorBody struct {
		Error cloudErrorBody `json:"error"`
	}
	b, err := io.ReadAll(body)
	if err != nil {
		return err.Error()
	}
	err = json.Unmarshal(b, &errorBody)
	if err != nil {
		return utils.ToSingleLine(string(b))
	}
	return fmt.Sprintf("%s: %s (target: %s)",
		errorBody.Error.Code, errorBody.Error.Message, errorBody.Error.Target)
}
