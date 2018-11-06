package main

import (
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListeningPort = "80"
	defaultRootURL       = "/"
	defaultDelay         = time.Duration(300)
)

var regexDomain = regexp.MustCompile(`^(([a-zA-Z]{1})|([a-zA-Z]{1}[a-zA-Z]{1})|([a-zA-Z]{1}[0-9]{1})|([0-9]{1}[a-zA-Z]{1})|([a-zA-Z0-9][a-zA-Z0-9-_]{1,61}[a-zA-Z0-9]))\.([a-zA-Z]{2,6}|[a-zA-Z0-9-]{2,30}\.[a-zA-Z]{2,3})$`).MatchString
var regexGodaddyKeySecret = regexp.MustCompile(`^[A-Za-z0-9]{12}\_[A-Za-z0-9]{22}\:[A-Za-z0-9]{22}$`).MatchString
var regexDuckDNSToken = regexp.MustCompile(`^[a-f0-9]{8}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{4}\-[a-f0-9]{12}$`).MatchString
var regexNamecheapPassword = regexp.MustCompile(`^[a-f0-9]{32}$`).MatchString

func parseEnvConfig() (listeningPort, rootURL string, delay time.Duration, updates []*updateType) {
	listeningPort = os.Getenv("LISTENINGPORT")
	if len(listeningPort) == 0 {
		listeningPort = defaultListeningPort
	} else {
		_, err := strconv.ParseInt(listeningPort, 10, 64)
		if err != nil {
			log.Fatal("LISTENINGPORT environment variable '" + listeningPort +
				"' is not a valid integer")
		}
	}
	rootURL = os.Getenv("ROOTURL")
	if len(rootURL) == 0 {
		rootURL = defaultRootURL
	} else if strings.ContainsAny(rootURL, " .?~#") {
		log.Fatal("ROOTURL environment variable '" + rootURL + "' contains invalid characters")
	}
	if rootURL[len(rootURL)-1] != '/' {
		rootURL += "/"
	}
	delayStr := os.Getenv("DELAY")
	if len(delayStr) == 0 {
		delay = defaultDelay
	} else {
		delayInt, err := strconv.ParseInt(delayStr, 10, 64)
		if err != nil {
			log.Fatal("DELAY environment variable '" + delayStr +
				"' is not a valid integer")
		}
		delay = time.Duration(delayInt)
	}
	var i uint64 = 1
	for {
		config := os.Getenv("RECORD" + strconv.FormatUint(i, 10))
		if config == "" {
			break
		}
		x := strings.Split(config, ",")
		if len(x) != 5 {
			log.Fatal("The configuration entry '" + config +
				"' should be in the format 'domain,host,provider,ipmethod,password'")
		}
		if !regexDomain(x[0]) {
			log.Fatal("The domain name '" + x[0] + "' is not valid for entry '" + config + "'")
		}
		if len(x[1]) == 0 {
			log.Fatal("The host for entry '" + config + "' must have one character at least")
		} // TODO test when it does not exist
		if x[2] == "duckdns" && x[1] != "@" {
			log.Fatal("The host '" + x[1] + "' can only be '@' for the DuckDNS entry '" + config + "'")
		}
		if x[2] != "namecheap" && x[2] != "godaddy" && x[2] != "duckdns" {
			log.Fatal("The DNS provider '" + x[2] + "' is not supported for entry '" + config + "'")
		}
		if x[2] == "namecheap" || x[2] == "duckdns" {
			if x[3] != "duckduckgo" && x[3] != "opendns" && regexIP(x[3]) == "" && x[3] != "provider" {
				log.Fatal("The IP query method '" + x[3] + "' is not valid for entry '" + config + "'")
			}
		} else {
			if x[3] != "duckduckgo" && x[3] != "opendns" && regexIP(x[3]) == "" {
				log.Fatal("The IP query method '" + x[3] + "' is not valid for entry '" + config + "'")
			}
		}

		if x[2] == "namecheap" && !regexNamecheapPassword(x[4]) {
			log.Fatal("The Namecheap password query parameter is not valid for entry '" + config + "'")
		}
		if x[2] == "godaddy" && !regexGodaddyKeySecret(x[4]) {
			log.Fatal("The GoDaddy password (key:secret) query parameter is not valid for entry '" + config + "'")
		}
		if x[2] == "duckdns" && !regexDuckDNSToken(x[4]) {
			log.Fatal("The DuckDNS password (token) query parameter is not valid for entry '" + config + "'")
		}
		var u updateType
		u.settings.domain = x[0]
		u.settings.host = x[1]
		u.settings.provider = x[2]
		u.settings.ipmethod = x[3]
		u.settings.password = x[4]
		updates = append(updates, &u)
		i++
	}
	if len(updates) == 0 {
		log.Fatal("No record to update was found in the environment variable RECORD1")
	}
	return listeningPort, rootURL, delay, updates
}
