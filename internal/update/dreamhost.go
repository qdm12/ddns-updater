package update

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/qdm12/ddns-updater/internal/constants"
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

func listDreamhostRecords(client network.Client, key string) (records dreamHostRecords, err error) {
	url := constants.DreamhostURL + "/?key=" + key + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-list_records"
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return records, err
	}
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

func removeDreamhostRecord(client network.Client, key, domain string, ip net.IP) error {
	url := constants.DreamhostURL + "?key=" + key + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-remove_record&record=" + strings.ToLower(domain) + "&type=A&value=" + ip.String()
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
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

func addDreamhostRecord(client network.Client, key, domain string, ip net.IP) error {
	url := constants.DreamhostURL + "?key=" + key + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-add_record&record=" + strings.ToLower(domain) + "&type=A&value=" + ip.String()
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
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
