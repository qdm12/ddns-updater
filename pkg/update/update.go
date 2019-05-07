package update

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"

	"ddns-updater/pkg/database"
	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/models"
	"ddns-updater/pkg/network"
	"ddns-updater/pkg/regex"

	uuid "github.com/google/uuid"
)

const (
	namecheapURL  = "https://dynamicdns.park-your-domain.com/update"
	godaddyURL    = "https://api.godaddy.com/v1/domains"
	duckdnsURL    = "https://www.duckdns.org/update"
	dreamhostURL  = "https://api.dreamhost.com"
	cloudflareURL = "https://api.cloudflare.com/client/v4"
)

func update(
	recordConfig *models.RecordConfigType,
	httpClient *http.Client,
	sqlDb *database.DB,
) {
	var err error
	recordConfig.M.Lock() // TODO hide through getters and setters
	defer recordConfig.M.Unlock()
	recordConfig.Status.Time = time.Now()

	// Get the public IP address
	ip, err := getPublicIP(httpClient, recordConfig.Settings.IPmethod)
	if err != nil {
		recordConfig.Status.Code = models.FAIL
		recordConfig.Status.Message = err.Error()
		logging.Warn("%s", recordConfig)
		return
	}
	// Note: empty IP means DNS provider provided
	if ip != "" && len(recordConfig.History.IPs) > 0 && ip == recordConfig.History.IPs[0] { // same IP as before
		recordConfig.Status.Code = models.UPTODATE
		recordConfig.Status.Message = "No IP change for " + time.Since(recordConfig.History.TSuccess).Round(time.Second).String()
		return
	}

	// Update the record
	if recordConfig.Settings.Provider == models.PROVIDERNAMECHEAP {
		ip, err = updateNamecheap(
			httpClient,
			recordConfig.Settings.Host,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Password,
			ip,
		)
	} else if recordConfig.Settings.Provider == models.PROVIDERGODADDY {
		err = updateGoDaddy(
			httpClient,
			recordConfig.Settings.Host,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Key,
			recordConfig.Settings.Secret,
			ip,
		)
	} else if recordConfig.Settings.Provider == models.PROVIDERDUCKDNS {
		ip, err = updateDuckDNS(
			httpClient,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Token,
			ip,
		)
	} else if recordConfig.Settings.Provider == models.PROVIDERDREAMHOST {
		err = updateDreamhost(
			httpClient,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Key,
			ip,
			recordConfig.Settings.BuildDomainName(),
		)
	} else if recordConfig.Settings.Provider == models.PROVIDERCLOUDFLARE {
		err = updateCloudflare(
			httpClient,
			recordConfig.Settings.ZoneIdentifier,
			recordConfig.Settings.Identifier,
			recordConfig.Settings.Host,
			recordConfig.Settings.Email,
			recordConfig.Settings.Key,
			recordConfig.Settings.UserServiceKey,
			ip,
			recordConfig.Settings.Proxied,
		)
	} else {
		err = fmt.Errorf("provider %s is not supported", recordConfig.Settings.Provider)
	}
	if err != nil {
		recordConfig.Status.Code = models.FAIL
		recordConfig.Status.Message = err.Error()
		logging.Warn("%s", recordConfig)
		return
	}
	if len(recordConfig.History.IPs) > 0 && ip == recordConfig.History.IPs[0] { // same IP
		recordConfig.Status.Code = models.UPTODATE
		recordConfig.Status.Message = "No IP change for " + time.Since(recordConfig.History.TSuccess).Round(time.Second).String()
		err = sqlDb.UpdateIPTime(recordConfig.Settings.Domain, recordConfig.Settings.Host, ip)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = "Cannot update database: " + err.Error()
		}
		return
	}
	// new IP
	recordConfig.Status.Code = models.SUCCESS
	recordConfig.Status.Message = ""
	recordConfig.History.TSuccess = time.Now()
	recordConfig.History.IPs = append([]string{ip}, recordConfig.History.IPs...)
	err = sqlDb.StoreNewIP(recordConfig.Settings.Domain, recordConfig.Settings.Host, ip)
	if err != nil {
		recordConfig.Status.Code = models.FAIL
		recordConfig.Status.Message = "Cannot update database: " + err.Error()
	}
}

func getPublicIP(httpClient *http.Client, IPmethod models.IPMethodType) (ip string, err error) {
	if IPmethod == models.IPMETHODPROVIDER {
		return "", nil
	} else if IPmethod == models.IPMETHODDUCKDUCKGO {
		return network.GetPublicIP(httpClient, "https://duckduckgo.com/?q=ip")
	} else if IPmethod == models.IPMETHODOPENDNS {
		return network.GetPublicIP(httpClient, "https://diagnostic.opendns.com/myip")
	}
	return "", fmt.Errorf("IPMethod %s is not supported", IPmethod)
}

func updateNamecheap(httpClient *http.Client, host, domain, password, ip string) (newIP string, err error) {
	url := strings.ToLower(namecheapURL + "?host=" + host + "&domain=" + domain + "&password=" + password)
	if len(ip) > 0 {
		url += "&ip=" + ip
	}
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	status, content, err := network.DoHTTPRequest(httpClient, r)
	if err != nil {
		return "", err
	}
	if status != 200 { // TODO test / combine with below
		return "", fmt.Errorf("%s responded with status %d", r.URL.String(), status)
	}
	var parsedXML struct {
		Errors struct {
			Error string `xml:"Err1"`
		} `xml:"errors"`
		IP string `xml:"IP"`
	}
	err = xml.Unmarshal(content, &parsedXML)
	if err != nil {
		return "", err
	}
	if parsedXML.Errors.Error != "" {
		return "", fmt.Errorf(parsedXML.Errors.Error)
	}
	if parsedXML.IP == "" {
		return "", fmt.Errorf("No IP address was sent back from DDNS server")
	}
	if regex.SearchIP(parsedXML.IP) == nil {
		return "", fmt.Errorf("IP address %s is not valid", parsedXML.IP)
	}
	return parsedXML.IP, nil
}

func updateGoDaddy(httpClient *http.Client, host, domain, key, secret, ip string) error {
	if len(ip) == 0 {
		return fmt.Errorf("cannot have a DNS provider-provided IP address for GoDaddy")
	}
	type goDaddyPutBody struct {
		Data string `json:"data"` // IP address to update to
	}
	URL := godaddyURL + "/" + strings.ToLower(domain) + "/records/A/" + strings.ToLower(host)
	r, err := network.BuildHTTPPut(
		URL,
		[]goDaddyPutBody{
			goDaddyPutBody{
				ip,
			},
		},
	)
	if err != nil {
		return err
	}
	r.Header.Set("Authorization", "sso-key "+key+":"+secret)
	status, content, err := network.DoHTTPRequest(httpClient, r)
	if err != nil {
		return err
	}
	if status != 200 {
		var parsedJSON struct {
			Message string `json:"message"`
		}
		err = json.Unmarshal(content, &parsedJSON)
		if err != nil {
			return err
		} else if parsedJSON.Message != "" {
			return fmt.Errorf("HTTP %d - %s", status, parsedJSON.Message)
		}
		return fmt.Errorf("HTTP %d", status)
	}
	return nil
}

func updateCloudflare(httpClient *http.Client, zoneIdentifier, identifier, host, email, key, userServiceKey, ip string, proxied bool) error {
	if len(ip) == 0 {
		return fmt.Errorf("cannot have a DNS provider-provided IP address for Cloudflare")
	}
	type cloudflarePutBody struct {
		Type    string `json:"type"`    // forced to A
		Name    string `json:"name"`    // DNS record name i.e. example.com
		Content string `json:"content"` // ip address
		Proxied bool   `json:"proxied"` // whether the record is receiving the performance and security benefits of Cloudflare
	}
	URL := cloudflareURL + "/zones/" + zoneIdentifier + "/dns_records/" + identifier
	r, err := network.BuildHTTPPut(
		URL,
		cloudflarePutBody{
			Type:    "A",
			Name:    host,
			Content: ip,
			Proxied: proxied,
		},
	)
	if err != nil {
		return err
	}
	if len(userServiceKey) > 0 {
		r.Header.Set("X-Auth-User-Service-Key", userServiceKey)
	} else if len(email) > 0 && len(key) > 0 {
		r.Header.Set("X-Auth-Email", email)
		r.Header.Set("X-Auth-Key", key)
	} else {
		return fmt.Errorf("email and key are both unset and no user service key was provided")
	}
	status, content, err := network.DoHTTPRequest(httpClient, r)
	if err != nil {
		return err
	}
	if status > 415 {
		return fmt.Errorf("bad HTTP status %d", status)
	}
	var parsedJSON struct {
		Success bool `json:"success"`
		Errors  []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
		Result struct {
			Content string `json:"content"`
		} `json:"result"`
	}
	err = json.Unmarshal(content, &parsedJSON)
	if err != nil {
		return err
	} else if !parsedJSON.Success {
		var errStr string
		for _, e := range parsedJSON.Errors {
			errStr += fmt.Sprintf("error %d: %s; ", e.Code, e.Message)
		}
		return fmt.Errorf(errStr)
	} else if parsedJSON.Result.Content != ip {
		return fmt.Errorf("returned IP address is %s and not %s", parsedJSON.Result.Content, ip)
	}
	return nil
}

func updateDuckDNS(httpClient *http.Client, domain, token, ip string) (newIP string, err error) {
	url := strings.ToLower(duckdnsURL + "?domains=" + domain + "&token=" + token + "&verbose=true")
	if len(ip) > 0 {
		url += "&ip=" + ip
	}
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	status, content, err := network.DoHTTPRequest(httpClient, r)
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("HTTP %d", status)
	}
	s := string(content)
	if s[0:2] == "KO" {
		return "", fmt.Errorf("Bad DuckDNS domain/token combination")
	} else if s[0:2] == "OK" {
		ips := regex.SearchIP(s)
		if ips == nil {
			return "", fmt.Errorf("DuckDNS did not respond with an IP address")
		}
		newIP = ips[0]
		return newIP, nil
	}
	return "", fmt.Errorf("DuckDNS responded with: %s", s)
}

func updateDreamhost(httpClient *http.Client, domain, key, ip, domainName string) error {
	type dreamhostReponse struct {
		Result string `json:"result"`
		Data   string `json:"data"`
	}
	// List records
	url := strings.ToLower(dreamhostURL + "/?key=" + key + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-list_records")
	r, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	status, content, err := network.DoHTTPRequest(httpClient, r)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("HTTP %d", status)
	}
	var dhList struct {
		Result string `json:"result"`
		Data   []struct {
			Editable string `json:"editable"`
			Type     string `json:"type"`
			Record   string `json:"record"`
			Value    string `json:"value"`
		} `json:"data"`
	}
	err = json.Unmarshal(content, &dhList)
	if err != nil {
		return err
	} else if dhList.Result != "success" {
		return fmt.Errorf(dhList.Result)
	}
	var found bool
	var oldIP string
	for _, data := range dhList.Data {
		if data.Type == "A" && data.Record == domainName {
			if data.Editable == "0" {
				return fmt.Errorf("Record data is not editable")
			}
			oldIP = data.Value
			if oldIP == ip { // success, nothing to change
				return nil
			}
			found = true
			break
		}
	}
	if found { // Found editable record with a different IP address, so remove it
		url = strings.ToLower(dreamhostURL + "?key=" + key + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-remove_record&record=" + domain + "&type=A&value=" + oldIP)
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			return err
		}
		status, content, err = network.DoHTTPRequest(httpClient, r)
		if err != nil {
			return err
		}
		if status != 200 {
			return fmt.Errorf("HTTP %d", status)
		}
		var dhResponse dreamhostReponse
		err = json.Unmarshal(content, &dhResponse)
		if err != nil {
			return err
		} else if dhResponse.Result != "success" { // this should not happen
			return fmt.Errorf("%s - %s", dhResponse.Result, dhResponse.Data)
		}
	}
	// Create the right record
	url = strings.ToLower(dreamhostURL + "?key=" + key + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-add_record&record=" + domain + "&type=A&value=" + ip)
	r, err = http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	status, content, err = network.DoHTTPRequest(httpClient, r)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("HTTP %d", status)
	}
	var dhResponse dreamhostReponse
	err = json.Unmarshal(content, &dhResponse)
	if err != nil {
		return err
	} else if dhResponse.Result != "success" {
		return fmt.Errorf("%s - %s", dhResponse.Result, dhResponse.Data)
	}
	return nil
}
