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

type NetcupClient struct {
	client         *http.Client
	ApiKey         string
	Password       string
	Session        string
	CustomerNumber string
	endpoint       string
}

func NewClient(customerNumber string, apikey, password string, url string) *NetcupClient {
	return &NetcupClient{
		CustomerNumber: customerNumber,
		ApiKey:         apikey,
		Password:       password,
		client:         http.DefaultClient,
		endpoint:       url,
	}
}

func (c *NetcupClient) do(ctx context.Context, req *NetcupRequest) (*NetcupResponse, error) {
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
	// s := string(b)

	var res NetcupResponse
	err = json.Unmarshal(b, &res)
	if err != nil {
		return nil, err
	}

	if !res.isSuccess() {
		return nil, errors.ErrBadHTTPStatus // TODO change error
	}

	return &res, nil
}

func (c *NetcupClient) Login(ctx context.Context) error {
	var params = NewParams()
	params.AddParam("apikey", c.ApiKey)
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

func (c *NetcupClient) InfoDNSRecords(ctx context.Context, domainname string) (*DNSRecordSet, error) {
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

func (c *NetcupClient) UpdateDNSRecords(ctx context.Context, domainname string, dnsRecordSet *DNSRecordSet) (*NetcupResponse, error) {
	params, err := c.addAuthParams(domainname)
	if err != nil {
		return nil, err
	}

	params.AddParam("dnsrecordset", dnsRecordSet)
	request := NewNetcupRequest("updateDnsRecords", params)

	response, err := c.do(ctx, request)
	if err != nil {
		return nil, err
	}

	return response, nil
}

func (c *NetcupClient) addAuthParams(domainname string) (*Params, error) {
	if c.Session == "" {
		return nil, errors.ErrNoSession
	}

	params := NewParams()
	params.AddParam("apikey", c.ApiKey)
	params.AddParam("apisessionid", c.Session)
	params.AddParam("customernumber", c.CustomerNumber)
	params.AddParam("domainname", domainname)

	return &params, nil
}

func (c *NetcupClient) GetRecordToUpdate(ctx context.Context, domain string, host string, ip netip.Addr) (*DNSRecord, error) {
	fmt.Println("Try to get infoDNSRecords")
	records, err := c.InfoDNSRecords(ctx, domain)
	if err != nil {
		return nil, err
	}
	fmt.Println("Found records: ")
	fmt.Println(records)

	recordType := constants.A
	if ip.Is6() {
		recordType = constants.AAAA
	}
	fmt.Println(records.GetRecordOccurences(host, recordType) > 1)
	if records.GetRecordOccurences(host, recordType) > 1 {
		return nil, errors.ErrListRecords // TODO change error
	}
	fmt.Println("searchedRecord: ")
	searchedRecord := records.GetRecord(host, recordType)
	fmt.Println(searchedRecord)
	if searchedRecord == nil {
		searchedRecord = NewDNSRecord(host, recordType, ip.String())
	}
	searchedRecord.Destination = ip.String()
	return searchedRecord, nil
}
