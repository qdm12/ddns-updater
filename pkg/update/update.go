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
	recordConfig.IsUpdating.Lock()
	defer recordConfig.IsUpdating.Unlock()
	recordConfig.Status.SetTime(time.Now())

	// Get the public IP address
	ip, err := getPublicIP(httpClient, recordConfig.Settings.IPmethod)
	if err != nil {
		recordConfig.Status.SetCode(models.FAIL)
		recordConfig.Status.SetMessage("%s", err)
		logging.Warn("%s", recordConfig)
		return
	}
	// Note: empty IP means DNS provider provided
	ips := recordConfig.History.GetIPs()
	if ip != "" && len(ips) > 0 && ip == ips[0] { // same IP as before
		recordConfig.Status.SetCode(models.UPTODATE)
		recordConfig.Status.SetMessage("No IP change for %s", recordConfig.History.GetTSuccessDuration())
		return
	}

	// Update the record
	switch recordConfig.Settings.Provider {
	case models.PROVIDERNAMECHEAP:
		ip, err = updateNamecheap(
			httpClient,
			recordConfig.Settings.Host,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Password,
			ip,
		)
	case models.PROVIDERGODADDY:
		err = updateGoDaddy(
			httpClient,
			recordConfig.Settings.Host,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Key,
			recordConfig.Settings.Secret,
			ip,
		)
	case models.PROVIDERDUCKDNS:
		ip, err = updateDuckDNS(
			httpClient,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Token,
			ip,
		)
	case models.PROVIDERDREAMHOST:
		err = updateDreamhost(
			httpClient,
			recordConfig.Settings.Domain,
			recordConfig.Settings.Key,
			ip,
			recordConfig.Settings.BuildDomainName(),
		)
	case models.PROVIDERCLOUDFLARE:
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
	default:
		err = fmt.Errorf("unsupported provider \"%s\"", recordConfig.Settings.Provider)
	}
	if err != nil {
		recordConfig.Status.SetCode(models.FAIL)
		recordConfig.Status.SetMessage("%s", err)
		logging.Warn("%s", recordConfig)
		return
	}
	if len(ips) > 0 && ip == ips[0] { // same IP
		recordConfig.Status.SetCode(models.UPTODATE)
		recordConfig.Status.SetMessage("No IP change for %s", recordConfig.History.GetTSuccessDuration())
		err = sqlDb.UpdateIPTime(recordConfig.Settings.Domain, recordConfig.Settings.Host, ip)
		if err != nil {
			recordConfig.Status.SetCode(models.FAIL)
			recordConfig.Status.SetMessage("Cannot update database: %s", err)
		}
		return
	}
	// new IP
	recordConfig.Status.SetCode(models.SUCCESS)
	recordConfig.Status.SetMessage("")
	recordConfig.History.SetTSuccess(time.Now())
	recordConfig.History.PrependIP(ip)
	err = sqlDb.StoreNewIP(recordConfig.Settings.Domain, recordConfig.Settings.Host, ip)
	if err != nil {
		recordConfig.Status.SetCode(models.FAIL)
		recordConfig.Status.SetMessage("Cannot update database: %s", err)
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
	return "", fmt.Errorf("IP method %s is not supported", IPmethod)
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
	} else if status != 200 { // TODO test / combine with below
		return "", fmt.Errorf("HTTP status %d", status)
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
	} else if parsedXML.Errors.Error != "" {
		return "", fmt.Errorf(parsedXML.Errors.Error)
	}
	ips := regex.SearchIP(parsedXML.IP)
	if ips == nil {
		return "", fmt.Errorf("no IP address in response")
	}
	newIP = ips[0]
	if len(ip) > 0 && ip != newIP {
		return "", fmt.Errorf("new IP address %s is not %s", newIP, ip)
	}
	return newIP, nil
}

func updateGoDaddy(httpClient *http.Client, host, domain, key, secret, ip string) error {
	if len(ip) == 0 {
		return fmt.Errorf("invalid empty IP address")
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
			return fmt.Errorf("HTTP status %d - %s", status, parsedJSON.Message)
		}
		return fmt.Errorf("HTTP status %d", status)
	}
	return nil
}

func updateCloudflare(httpClient *http.Client, zoneIdentifier, identifier, host, email, key, userServiceKey, ip string, proxied bool) (err error) {
	if len(ip) == 0 {
		return fmt.Errorf("invalid empty IP address")
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
		return fmt.Errorf("HTTP status %d", status)
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
	newIP := parsedJSON.Result.Content
	if err != nil {
		return err
	} else if !parsedJSON.Success {
		var errStr string
		for _, e := range parsedJSON.Errors {
			errStr += fmt.Sprintf("error %d: %s; ", e.Code, e.Message)
		}
		return fmt.Errorf(errStr)
	} else if newIP != ip {
		return fmt.Errorf("new IP address %s is not %s", newIP, ip)
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
		return "", fmt.Errorf("HTTP status %d", status)
	}
	s := string(content)
	if s[0:2] == "KO" {
		return "", fmt.Errorf("invalid domain token combination")
	} else if s[0:2] == "OK" {
		ips := regex.SearchIP(s)
		if ips == nil {
			return "", fmt.Errorf("no IP address in response")
		}
		newIP = ips[0]
		if len(ip) > 0 && newIP != ip {
			return "", fmt.Errorf("new IP address %s is not %s", newIP, ip)
		}
		return newIP, nil
	}
	return "", fmt.Errorf("response \"%s\"", s)
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
		return fmt.Errorf("HTTP status %d", status)
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
				return fmt.Errorf("record data is not editable")
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
			return fmt.Errorf("HTTP status %d", status)
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
		return fmt.Errorf("HTTP status %d", status)
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
