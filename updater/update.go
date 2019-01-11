package main

import (
	"encoding/json"
	"encoding/xml"
	"log"
	"net/http"
	"strings"
	"time"
)

const (
	namecheapURL = "https://dynamicdns.park-your-domain.com/update"
	godaddyURL   = "https://api.godaddy.com/v1/domains"
	duckdnsURL   = "https://www.duckdns.org/update"
)

type GoDaddyPutBody struct {
	Data string `json:"data"` // IP address to update to
}

func buildRequest(host, domain, provider, password, ip string) (r *http.Request, err error) {
	if provider == "namecheap" {
		url := namecheapURL + "?host=" + strings.ToLower(host) +
			"&domain=" + strings.ToLower(domain) + "&password=" + strings.ToLower(password)
		if ip != "provider" {
			url += "&ip=" + ip
		}
		r, err = buildHTTPGet(url)
		if err != nil {
			return nil, err
		}
	} else if provider == "godaddy" {
		url := godaddyURL + "/" + strings.ToLower(domain) + "/records/A/" + strings.ToLower(host)
		r, err = buildHTTPPutJSONAuth(
			url,
			"sso-key "+password, // password is key:secret here
			[]GoDaddyPutBody{
				GoDaddyPutBody{
					ip,
				},
			},
		)
		if err != nil {
			return nil, err
		}
	} else { // duckdns
		url := duckdnsURL + "?domains=" + strings.ToLower(domain) +
			"&token=" + password + "&verbose=true"
		if ip != "provider" {
			url += "&ip=" + ip
		}
		r, err = buildHTTPGet(url)
		if err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (u *updateType) update() {
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
		ip, err = getPublicIP("https://duckduckgo.com/?q=ip")
		if err != nil {
			u.status.code = FAIL
			u.status.message = err.Error()
			log.Println(u.String())
			return
		}
	} else if u.settings.ipmethod == "opendns" {
		ip, err = getPublicIP("https://diagnostic.opendns.com/myip")
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

	// Build the dynamic DNS request to update the IP address
	req, err := buildRequest(u.settings.host, u.settings.domain, u.settings.provider, u.settings.password, ip)
	if err != nil {
		u.status.code = FAIL
		u.status.message = err.Error()
		log.Println(u.String())
		return
	}
	status, content, err := doHTTPRequest(req, httpGetTimeout)
	if err != nil {
		u.status.code = FAIL
		u.status.message = err.Error()
		log.Println(u.String())
		return
	}
	if u.settings.provider == "namecheap" {
		if status != "200" { // TODO test / combine with below
			u.status.code = FAIL
			u.status.message = req.URL.String() + " responded with status " + status
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
	} else { // duckdns
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
	}
	if len(u.extras.ips) > 0 && ip == u.extras.ips[0] { // same IP
		u.status.code = UPTODATE
		u.status.message = "No IP change for " + time.Since(u.extras.tSuccess).Round(time.Second).String()
		return
	}
	u.status.code = SUCCESS
	u.status.message = ""
	u.extras.tSuccess = time.Now()
	u.extras.ips = append([]string{ip}, u.extras.ips...)
}
