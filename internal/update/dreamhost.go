package update

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/qdm12/golibs/network"
)

const success = "success"

type (
	dreamHostRecords struct {
		Result string `json:"result"`
		Data   []struct {
			Editable string `json:"editable"`
			Type     string `json:"type"`
			Record   string `json:"record"`
			Value    string `json:"value"`
		} `json:"data"`
	}
	dreamhostReponse struct {
		Result string `json:"result"`
		Data   string `json:"data"`
	}
)

func makeDreamhostDefaultValues(key string) (values url.Values) { //nolint:unparam
	values.Set("key", key)
	values.Set("unique_id", uuid.New().String())
	values.Set("format", "json")
	return values
}

func listDreamhostRecords(client network.Client, key string) (records dreamHostRecords, err error) {
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-list_records")
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return records, err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return records, err
	} else if status != http.StatusOK {
		return records, fmt.Errorf("HTTP status %d", status)
	}
	if err := json.Unmarshal(content, &records); err != nil {
		return records, err
	} else if records.Result != success {
		return records, fmt.Errorf(records.Result)
	}
	return records, nil
}

func removeDreamhostRecord(client network.Client, key, domain string, ip net.IP) error { //nolint:dupl
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-remove_record")
	values.Set("record", domain)
	values.Set("type", "A")
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	var dhResponse dreamhostReponse
	if err := json.Unmarshal(content, &dhResponse); err != nil {
		return err
	} else if dhResponse.Result != success { // this should not happen
		return fmt.Errorf("%s - %s", dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func addDreamhostRecord(client network.Client, key, domain string, ip net.IP) error { //nolint:dupl
	u := url.URL{
		Scheme: "https",
		Host:   "api.dreamhost.com",
	}
	values := makeDreamhostDefaultValues(key)
	values.Set("cmd", "dns-add_record")
	values.Set("record", domain)
	values.Set("type", "A")
	values.Set("value", ip.String())
	u.RawQuery = values.Encode()
	r, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	r.Header.Set("User-Agent", "DDNS-Updater quentin.mcgaw@gmail.com")
	status, content, err := client.DoHTTPRequest(r)
	if err != nil {
		return err
	} else if status != http.StatusOK {
		return fmt.Errorf("HTTP status %d", status)
	}
	var dhResponse dreamhostReponse
	if err := json.Unmarshal(content, &dhResponse); err != nil {
		return err
	} else if dhResponse.Result != success {
		return fmt.Errorf("%s - %s", dhResponse.Result, dhResponse.Data)
	}
	return nil
}

func updateDreamhost(client network.Client, domain, key, domainName string, ip net.IP) error {
	if ip == nil {
		return fmt.Errorf("IP address was not given to updater")
	}
	records, err := listDreamhostRecords(client, key)
	if err != nil {
		return err
	}
	var oldIP net.IP
	for _, data := range records.Data {
		if data.Type == "A" && data.Record == domainName {
			if data.Editable == "0" {
				return fmt.Errorf("record data is not editable")
			}
			oldIP = net.ParseIP(data.Value)
			if ip.Equal(oldIP) { // success, nothing to change
				return nil
			}
			break
		}
	}
	if oldIP != nil { // Found editable record with a different IP address, so remove it
		if err := removeDreamhostRecord(client, key, domain, oldIP); err != nil {
			return err
		}
	}
	return addDreamhostRecord(client, key, domain, ip)
}
