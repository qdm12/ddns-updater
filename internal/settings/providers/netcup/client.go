package netcup

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/netip"

	"github.com/qdm12/ddns-updater/internal/settings/constants"
	"github.com/qdm12/ddns-updater/internal/settings/errors"
	"github.com/qdm12/ddns-updater/internal/settings/headers"
	"golang.org/x/net/context"
)

type Client struct {
	client         *http.Client
	APIKey         string
	Password       string
	Session        string
	CustomerNumber string
	endpoint       string
}

func NewClient(ctx context.Context, customerNumber, apikey, password, url string) *Client {
	return &Client{
		CustomerNumber: customerNumber,
		APIKey:         apikey,
		Password:       password,
		client:         http.DefaultClient,
		endpoint:       url,
	}
}

func (c *Client) do(ctx context.Context, req *Request) (*Response, error) {
	b, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewBuffer(b))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrBadRequest, err)
	}
	headers.SetUserAgent(request)
	response, err := c.client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnsuccessfulResponse, err)
	}
	defer response.Body.Close()

	b, err = io.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errors.ErrUnmarshalResponse, err)
	}

	var res Response
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}

	if res.isError() {
		return nil, errors.ErrBadHTTPStatus // TODO change error
	}

	return &res, nil
}

func (c *Client) Login(ctx context.Context) error {
	var params = NewParams()
	params.AddParam("apikey", c.APIKey)
	params.AddParam("apipassword", c.Password)
	params.AddParam("customernumber", c.CustomerNumber)

	request := NewNetcupRequest("login", &params)

	response, err := c.do(ctx, request)
	if err != nil {
		return err
	}

	var loginResponse LoginResponse
	err = json.Unmarshal(response.ResponseData, &loginResponse)

	switch {
	case err != nil:
		return err
	case loginResponse.Session == "":
		return errors.ErrNoSession
	default:
		c.Session = loginResponse.Session
	}

	return nil
}

func (c *Client) Logout() error {
	return nil
}

func (c *Client) InfoDNSRecords(ctx context.Context, domainname string) (*DNSRecordSet, error) {
	params, err := c.addAuthParams(domainname)
	if err != nil {
		return nil, err
	}

	request := NewNetcupRequest("infoDnsRecords", params)

	response, err := c.do(ctx, request)
	if err != nil {
		return nil, err
	}

	var dnsRecordSet DNSRecordSet
	err = json.Unmarshal(response.ResponseData, &dnsRecordSet)
	if err != nil {
		return nil, err
	}

	return &dnsRecordSet, nil
}

func (c *Client) UpdateDNSRecords(ctx context.Context, domain string, records *DNSRecordSet) (*Response, error) {
	params, err := c.addAuthParams(domain)
	if err != nil {
		return nil, err
	}

	params.AddParam("dnsrecordset", records)
	request := NewNetcupRequest("updateDnsRecords", params)

	response, err := c.do(ctx, request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *Client) addAuthParams(domainname string) (*Params, error) {
	if c.Session == "" {
		return nil, errors.ErrNoSession
	}

	params := NewParams()
	params.AddParam("apikey", c.APIKey)
	params.AddParam("apisessionid", c.Session)
	params.AddParam("customernumber", c.CustomerNumber)
	params.AddParam("domainname", domainname)

	return &params, nil
}

func (c *Client) GetRecordToUpdate(ctx context.Context, domain, host string, ip netip.Addr) (*DNSRecord, error) {
	records, err := c.InfoDNSRecords(ctx, domain)
	if err != nil {
		return nil, err
	}

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	if records.GetRecordOccurences(host, recordType) > 1 {
		return nil, errors.ErrListRecords
	}

	searchedRecord := records.GetRecord(host, recordType)
	if searchedRecord == nil {
		searchedRecord = NewDNSRecord(host, recordType, ip.String())
	}
	searchedRecord.Destination = ip.String()
	return searchedRecord, nil
}
