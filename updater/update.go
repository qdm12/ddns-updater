package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"strings"
	"time"

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

// i is the index of the update to update
func (env *envType) update(i int) {
	u := &env.updates[i]
	u.m.Lock()
	defer u.m.Unlock()
	if u.status.code == UPDATING {
		log.Println(u.String())
		return
	}
	u.status.code = UPDATING
	defer func() {
		if u.status.code == UPDATING {
			u.status.code = FAIL
			u.status.message = "Status not changed from UPDATING"
		}
	}()
	u.status.time = time.Now()

	// Get the public IP address
	var ip string
	var err error
	if u.settings.ipmethod == "provider" {
		ip = ""
	} else if u.settings.ipmethod == "duckduckgo" {
		ip, err = getPublicIP(env.httpClient, "https://duckduckgo.com/?q=ip")
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
	} else if u.settings.ipmethod == "opendns" {
		ip, err = getPublicIP(env.httpClient, "https://diagnostic.opendns.com/myip")
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
	} else { // fixed IP
		ip = u.settings.ipmethod
	}
	if ip != "" && len(u.extras.ips) > 0 && ip == u.extras.ips[0] { // same IP
		u.status.code = UPTODATE
		u.status.message = "No IP change for " + time.Since(u.extras.tSuccess).Round(time.Second).String()
		return
	}

	// Update the record
	if u.settings.provider == "namecheap" {
		url := namecheapURL + "?host=" + strings.ToLower(u.settings.host) +
			"&domain=" + strings.ToLower(u.settings.domain) + "&password=" + strings.ToLower(u.settings.password)
		if ip != "provider" {
			url += "&ip=" + ip
		}
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		status, content, err := doHTTPRequest(env.httpClient, r)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		if status != "200" { // TODO test / combine with below
			u.status.code = FAIL
			u.status.message = r.URL.String() + " responded with status " + status
			log.Println(u.String())
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
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		if parsedXML.Errors.Error != "" {
			u.status.code = FAIL
			u.status.message = parsedXML.Errors.Error
			log.Println(u.String())
			return
		}
		if parsedXML.IP == "" {
			u.status.code = FAIL
			u.status.message = "No IP address was sent back from DDNS server"
			log.Println(u.String())
			return
		}
		if regexIP(parsedXML.IP) == "" {
			u.status.code = FAIL
			u.status.message = "IP address " + parsedXML.IP + " is not valid"
			log.Println(u.String())
			return
		}
		ip = parsedXML.IP
	} else if u.settings.provider == "godaddy" {
		url := godaddyURL + "/" + strings.ToLower(u.settings.domain) + "/records/A/" + strings.ToLower(u.settings.host)
		r, err := buildHTTPPutJSONAuth(
			url,
			"sso-key "+u.settings.password, // password is key:secret here
			[]goDaddyPutBody{
				goDaddyPutBody{
					ip,
				},
			},
		)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		status, content, err := doHTTPRequest(env.httpClient, r)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		if status != "200" {
			u.status.code = FAIL
			u.status.message = "HTTP " + status
			var parsedJSON struct {
				Message string `json:"message"`
			}
			err = json.Unmarshal(content, &parsedJSON)
			if err != nil {
				u.status.message = err.Error()
			} else if parsedJSON.Message != "" {
				u.status.message += " - " + parsedJSON.Message
			}
			log.Println(u.String())
			return
		}
	} else if u.settings.provider == "duckdns" {
		url := duckdnsURL + "?domains=" + strings.ToLower(u.settings.domain) +
			"&token=" + u.settings.password + "&verbose=true"
		if ip != "provider" {
			url += "&ip=" + ip
		}
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		status, content, err := doHTTPRequest(env.httpClient, r)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		if status != "200" {
			u.status.code = FAIL
			u.status.message = "HTTP " + status
			log.Println(u.String())
			return
		}
		s := string(content)
		if s[0:2] == "KO" {
			u.status.code = FAIL
			u.status.message = "Bad DuckDNS domain/token combination"
			log.Println(u.String())
			return
		} else if s[0:2] == "OK" {
			ip = regexIP(s)
			if ip == "" {
				u.status.code = FAIL
				u.status.message = "DuckDNS did not respond with an IP address"
				log.Println(u.String())
				return
			}
		} else {
			u.status.code = FAIL
			u.status.message = "DuckDNS responded with '" + s + "'"
			log.Println(u.String())
			return
		}
	} else if u.settings.provider == "dreamhost" {
		url := dreamhostURL + "/?key=" + u.settings.password + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-list_records"
		r, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		status, content, err := doHTTPRequest(env.httpClient, r)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		if status != "200" {
			u.status.code = FAIL
			u.status.message = "HTTP " + status
			log.Println(u.String())
			return
		}
		var dhList dreamhostList
		err = json.Unmarshal(content, &dhList)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		} else if dhList.Result != "success" {
			u.status.code = FAIL
			u.status.message = dhList.Result
			log.Println(u.String())
			return
		}
		var oldIP string
		var found bool
		for _, data := range dhList.Data {
			if data.Type == "A" && data.Record == u.settings.buildDomainName() {
				if data.Editable == "0" {
					u.status.code = FAIL
					u.status.message = "Record data is not editable"
					log.Println(u.String())
					return
				}
				oldIP := data.Value
				if oldIP == ip {
					u.status.code = UPTODATE
					u.status.message = "No IP change for " + time.Since(u.extras.tSuccess).Round(time.Second).String()
					return
				}
				found = true
				break
			}
		}
		if found {
			url = dreamhostURL + "?key=" + u.settings.password + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-remove_record&record=" + strings.ToLower(u.settings.domain) + "&type=A&value=" + oldIP
			r, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				u.status.code = FAIL
				u.status.message = err.Error()
				log.Println(u.String())
				return
			}
			status, content, err = doHTTPRequest(env.httpClient, r)
			if err != nil {
				u.status.code = FAIL
				u.status.message = err.Error()
				log.Println(u.String())
				return
			}
			if status != "200" {
				u.status.code = FAIL
				u.status.message = "HTTP " + status
				log.Println(u.String())
				return
			}
			var dhResponse dreamhostReponse
			err = json.Unmarshal(content, &dhResponse)
			if err != nil {
				u.status.code = FAIL
				u.status.message = err.Error()
				log.Println(u.String())
				return
			} else if dhResponse.Result != "success" { // this should not happen
				u.status.code = FAIL
				u.status.message = dhResponse.Result + " - " + dhResponse.Data
				log.Println(u.String())
				return
			}
		}
		url = dreamhostURL + "?key=" + u.settings.password + "&unique_id=" + uuid.New().String() + "&format=json&cmd=dns-add_record&record=" + strings.ToLower(u.settings.domain) + "&type=A&value=" + ip
		r, err = http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		status, content, err = doHTTPRequest(env.httpClient, r)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
		if status != "200" {
			u.status.code = FAIL
			u.status.message = "HTTP " + status
			log.Println(u.String())
			return
		}
		var dhResponse dreamhostReponse
		err = json.Unmarshal(content, &dhResponse)
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		} else if dhResponse.Result != "success" {
			u.status.code = FAIL
			u.status.message = dhResponse.Result + " - " + dhResponse.Data
			log.Println(u.String())
			return
		}
	}
	if len(u.extras.ips) > 0 && ip == u.extras.ips[0] { // same IP
		u.status.code = UPTODATE
		u.status.message = "No IP change for " + time.Since(u.extras.tSuccess).Round(time.Second).String()
		err = env.dbContainer.updateIPTime(u.settings.domain, u.settings.host, ip)
		if err != nil {
			u.status.code = FAIL
			u.status.message = "Cannot update database: " + err.Error()
		}
		return
	}
	// new IP
	u.status.code = SUCCESS
	u.status.message = ""
	u.extras.tSuccess = time.Now()
	u.extras.ips = append([]string{ip}, u.extras.ips...)
	err = env.dbContainer.storeNewIP(u.settings.domain, u.settings.host, ip)
	if err != nil {
		u.status.code = FAIL
		u.status.message = "Cannot update database: " + err.Error()
	}
}
