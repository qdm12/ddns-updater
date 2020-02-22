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
	body := bytes.NewBufferString(url.Values{
		"login_token": []string{token},
		"format":      []string{"json"},
		"domain":      []string{domain},
		"length":      []string{"200"},
		"sub_domain":  []string{host},
		"record_type": []string{"A"},
	}.Encode())
	req, err := http.NewRequest(http.MethodPost, "https://dnsapi.cn/Record.List", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	status, content, err := client.DoHTTPRequest(req)
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
	body = bytes.NewBufferString(url.Values{
		"login_token": []string{token},
		"format":      []string{"json"},
		"domain":      []string{domain},
		"record_id":   []string{recordID},
		"value":       []string{ip.String()},
		"record_line": []string{recordLine},
		"sub_domain":  []string{host},
	}.Encode())
	req, err = http.NewRequest(http.MethodPost, "https://dnsapi.cn/Record.Ddns", body)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	status, content, err = client.DoHTTPRequest(req)
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
