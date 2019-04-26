package update

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"strings"
	"time"
	"fmt"

	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/database"
	"ddns-updater/pkg/models"
	"ddns-updater/pkg/network"
	"ddns-updater/pkg/regex"
	uuid "github.com/google/uuid"
)

const (
	namecheapURL = "https://dynamicdns.park-your-domain.com/update"
	godaddyURL   = "https://api.godaddy.com/v1/domains"
	duckdnsURL   = "https://www.duckdns.org/update"
	dreamhostURL = "https://api.dreamhost.com"
)

type goDaddyPutBody struct {
	Data string `json:"data"` // IP address to update to
}

type dreamhostList struct {
	Result string          `json:"result"`
	Data   []dreamhostData `json:"data"`
}

type dreamhostData struct {
	Editable string `json:"editable"`
	Type     string `json:"type"`
	Record   string `json:"record"`
	Value    string `json:"value"`
}

type dreamhostReponse struct {
	Result string `json:"result"`
	Data   string `json:"data"`
}

func update(
	recordConfig *models.RecordConfigType,
	httpClient *http.Client,
	sqlDb *database.DB,
) {
	recordConfig.Lock()
	defer recordConfig.Unlock()
	if recordConfig.Status.Code == models.UPDATING {
		logging.Info(recordConfig.String())
		return
	}
	recordConfig.Status.Code = models.UPDATING
	defer func() {
		if recordConfig.Status.Code == models.UPDATING {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = "Status not changed from UPDATING"
		}
	}()
	recordConfig.Status.Time = time.Now()

	// Get the public IP address
	var ip string
	var err error
	if recordConfig.Settings.IPmethod == "provider" {
		ip = ""
	} else if recordConfig.Settings.IPmethod == "duckduckgo" {
		ip, err = network.GetPublicIP(httpClient, "https://duckduckgo.com/?q=ip")
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
	} else if recordConfig.Settings.IPmethod == "opendns" {
		ip, err = network.GetPublicIP(httpClient, "https://diagnostic.opendns.com/myip")
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
	} else { // fixed IP
		ip = recordConfig.Settings.IPmethod
	}
	if ip != "" && len(recordConfig.History.IPs) > 0 && ip == recordConfig.History.IPs[0] { // same IP
		recordConfig.Status.Code = models.UPTODATE
		recordConfig.Status.Message = "No IP change for " + time.Since(recordConfig.History.TSuccess).Round(time.Second).String()
		return
	}

	// Update the record
	if recordConfig.Settings.Provider == "namecheap" {
		url := namecheapURL + "?host=" + strings.ToLower(recordConfig.Settings.Host) +
			"&domain=" + strings.ToLower(recordConfig.Settings.Domain) + "&password=" + strings.ToLower(recordConfig.Settings.Password)
		if ip != "provider" {
			url += "&ip=" + ip
		}
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		status, content, err := network.DoHTTPRequest(httpClient, r)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		if status != 200 { // TODO test / combine with below
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = fmt.Sprintf("%s responded with status %d", r.URL.String(), status)
			log.Println(recordConfig.String())
			return
		}
		var parsedXML struct {
			Errors struct {
				Error string `xml:"Err1"`
			} `xml:"errors"`
			IP string `xml:"IP"`
		}
		err = xml.Unmarshal(content, &parsedXML)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		if parsedXML.Errors.Error != "" {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = parsedXML.Errors.Error
			log.Println(recordConfig.String())
			return
		}
		if parsedXML.IP == "" {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = "No IP address was sent back from DDNS server"
			log.Println(recordConfig.String())
			return
		}
		if regex.FindIP(parsedXML.IP) == "" {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = "IP address " + parsedXML.IP + " is not valid"
			log.Println(recordConfig.String())
			return
		}
		ip = parsedXML.IP
	} else if recordConfig.Settings.Provider == "godaddy" {
		url := godaddyURL + "/" + strings.ToLower(recordConfig.Settings.Domain) + "/records/A/" + strings.ToLower(recordConfig.Settings.Host)
		r, err := network.BuildHTTPPutJSONAuth(
			url,
			"sso-key "+recordConfig.Settings.Password, // password is key:secret here
			[]goDaddyPutBody{
				goDaddyPutBody{
					ip,
				},
			},
		)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		status, content, err := network.DoHTTPRequest(httpClient, r)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		if status != 200 {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = fmt.Sprintf("HTTP %d", status)
			var parsedJSON struct {
				Message string `json:"message"`
			}
			err = json.Unmarshal(content, &parsedJSON)
			if err != nil {
				recordConfig.Status.Message = err.Error()
			} else if parsedJSON.Message != "" {
				recordConfig.Status.Message += " - " + parsedJSON.Message
			}
			log.Println(recordConfig.String())
			return
		}
	} else if recordConfig.Settings.Provider == "duckdns" {
		url := duckdnsURL + "?domains=" + strings.ToLower(recordConfig.Settings.Domain) +
			"&token=" + recordConfig.Settings.Password + "&verbose=true"
		if ip != "provider" {
			url += "&ip=" + ip
		}
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		status, content, err := network.DoHTTPRequest(httpClient, r)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		if status != 200 {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = fmt.Sprintf("HTTP %d", status)
			log.Println(recordConfig.String())
			return
		}
		s := string(content)
		if s[0:2] == "KO" {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = "Bad DuckDNS domain/token combination"
			log.Println(recordConfig.String())
			return
		} else if s[0:2] == "OK" {
			ip = regex.FindIP(s)
			if ip == "" {
				recordConfig.Status.Code = models.FAIL
				recordConfig.Status.Message = "DuckDNS did not respond with an IP address"
				log.Println(recordConfig.String())
				return
			}
		} else {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = "DuckDNS responded with '" + s + "'"
			log.Println(recordConfig.String())
			return
		}
	} else if recordConfig.Settings.Provider == "dreamhost" {
		url := dreamhostURL + "/?key=" + recordConfig.Settings.Password + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-list_records"
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		status, content, err := network.DoHTTPRequest(httpClient, r)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		if status != 200 {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = fmt.Sprintf("HTTP %d", status)
			log.Println(recordConfig.String())
			return
		}
		var dhList dreamhostList
		err = json.Unmarshal(content, &dhList)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		} else if dhList.Result != "success" {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = dhList.Result
			log.Println(recordConfig.String())
			return
		}
		var oldIP string
		var found bool
		for _, data := range dhList.Data {
			if data.Type == "A" && data.Record == recordConfig.Settings.BuildDomainName() {
				if data.Editable == "0" {
					recordConfig.Status.Code = models.FAIL
					recordConfig.Status.Message = "Record data is not editable"
					log.Println(recordConfig.String())
					return
				}
				oldIP := data.Value
				if oldIP == ip {
					recordConfig.Status.Code = models.UPTODATE
					recordConfig.Status.Message = "No IP change for " + time.Since(recordConfig.History.TSuccess).Round(time.Second).String()
					return
				}
				found = true
				break
			}
		}
		if found {
			url = dreamhostURL + "?key=" + recordConfig.Settings.Password + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-remove_record&record=" + strings.ToLower(recordConfig.Settings.Domain) + "&type=A&value=" + oldIP
			r, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				recordConfig.Status.Code = models.FAIL
				recordConfig.Status.Message = err.Error()
				log.Println(recordConfig.String())
				return
			}
			status, content, err = network.DoHTTPRequest(httpClient, r)
			if err != nil {
				recordConfig.Status.Code = models.FAIL
				recordConfig.Status.Message = err.Error()
				log.Println(recordConfig.String())
				return
			}
			if status != 200 {
				recordConfig.Status.Code = models.FAIL
				recordConfig.Status.Message = fmt.Sprintf("HTTP %d", status)
				log.Println(recordConfig.String())
				return
			}
			var dhResponse dreamhostReponse
			err = json.Unmarshal(content, &dhResponse)
			if err != nil {
				recordConfig.Status.Code = models.FAIL
				recordConfig.Status.Message = err.Error()
				log.Println(recordConfig.String())
				return
			} else if dhResponse.Result != "success" { // this should not happen
				recordConfig.Status.Code = models.FAIL
				recordConfig.Status.Message = dhResponse.Result + " - " + dhResponse.Data
				log.Println(recordConfig.String())
				return
			}
		}
		url = dreamhostURL + "?key=" + recordConfig.Settings.Password + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-add_record&record=" + strings.ToLower(recordConfig.Settings.Domain) + "&type=A&value=" + ip
		r, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		status, content, err = network.DoHTTPRequest(httpClient, r)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		}
		if status != 200 {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = fmt.Sprintf("HTTP %d", status)
			log.Println(recordConfig.String())
			return
		}
		var dhResponse dreamhostReponse
		err = json.Unmarshal(content, &dhResponse)
		if err != nil {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = err.Error()
			log.Println(recordConfig.String())
			return
		} else if dhResponse.Result != "success" {
			recordConfig.Status.Code = models.FAIL
			recordConfig.Status.Message = dhResponse.Result + " - " + dhResponse.Data
			log.Println(recordConfig.String())
			return
		}
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
