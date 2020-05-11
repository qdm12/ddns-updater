package update

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/qdm12/golibs/network"
)

func updateDNSPod(client network.Client, domain, host, token string, ip net.IP) (err error) {
	if ip == nil {
		return fmt.Errorf("IP address was not given to updater")
	}
	u := url.URL{
		Scheme: "https",
		Host:   "dnsapi.cn",
		Path:   "/Record.List",
	}
	var values url.Values
	values.Set("login_token", token)
	values.Set("format", "json")
	values.Set("domain", domain)
	values.Set("length", "200")
	values.Set("sub_domain", host)
	values.Set("record_type", "A")
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodPost, u.String(), bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	var recordResp struct {
		Records []struct {
			ID    string `json:"id"`
			Value string `json:"value"`
			Type  string `json:"type"`
			Name  string `json:"name"`
			Line  string `json:"line"`
		} `json:"records"`
	}
	if err := json.Unmarshal(content, &recordResp); err != nil {
		return err
	}
	var recordID, recordLine string
	for _, record := range recordResp.Records {
		if record.Type == "A" && record.Name == host {
			receivedIP := net.ParseIP(record.Value)
			if ip.Equal(receivedIP) {
				return nil
			}
			recordID = record.ID
			recordLine = record.Line
			break
		}
	}
	if len(recordID) == 0 {
		return fmt.Errorf("record not found")
	}

	u.Path = "/Record.Ddns"
	values = url.Values{}
	values.Set("login_token", token)
	values.Set("format", "json")
	values.Set("domain", domain)
	values.Set("record_id", recordID)
	values.Set("value", ip.String())
	values.Set("record_line", recordLine)
	values.Set("sub_domain", host)
	u.RawQuery = values.Encode()
	r, err = http.NewRequest(http.MethodPost, u.String(), bytes.NewBufferString(values.Encode()))
	if err != nil {
		return err
	}
	r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err = client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	var ddnsResp struct {
		Record struct {
			ID    int64  `json:"id"`
			Value string `json:"value"`
			Name  string `json:"name"`
		} `json:"record"`
	}
	if err := json.Unmarshal(content, &ddnsResp); err != nil {
		return err
	}
	receivedIP := net.ParseIP(ddnsResp.Record.Value)
	if !ip.Equal(receivedIP) {
		return fmt.Errorf("ip not set")
	}
	return nil
}
