package main

import (
	"strconv"
	"time"
)

type HTMLData struct {
	Updates []UpdateType
}

type UpdateType struct { // exported for HTML template
	Domain   string
	Host     string
	Provider string
	IPMethod string
	Status   string
	IP       string   // current set ip
	IPs      []string // previous ips
}

func durationString(t time.Time) (durationStr string) {
	duration := time.Since(t)
	if duration < time.Minute {
		return strconv.FormatFloat(duration.Round(time.Second).Seconds(), 'f', -1, 64) + "s"
	} else if duration < time.Hour {
		return strconv.FormatFloat(duration.Round(time.Minute).Minutes(), 'f', -1, 64) + "m"
	} else if duration < 24*time.Hour {
		return strconv.FormatFloat(duration.Round(time.Hour).Hours(), 'f', -1, 64) + "h"
	} else {
		return strconv.FormatFloat(duration.Round(time.Hour*24).Hours()/24, 'f', -1, 64) + "d"
	}
}

func updatesToHTML(updates *Updates) (htmlData *HTMLData) {
	htmlData = new(HTMLData)
	for _, u := range *updates {
		var U UpdateType
		U.Domain = u.settings.htmlDomain()
		U.Host = u.settings.host // TODO html method
		U.Provider = u.settings.htmlProvider()
		U.IPMethod = u.settings.htmlIpmethod()
		if u.status.code == UPTODATE {
			u.status.message = "No IP change for " + durationString(u.extras.tSuccess)
		}
		U.Status = u.status.html()
		if len(u.extras.ips) > 0 {
			U.IP = "<a href=\"https://ipinfo.io/" + u.extras.ips[0] + "\">" + u.extras.ips[0] + "</a>"
		} else {
			U.IP = "N/A"
		}
		if len(u.extras.ips) > 1 {
			U.IPs = u.extras.ips[1:]
			for i := range U.IPs {
				if i == len(U.IPs)-1 {
					break
				}
				U.IPs[i] += ", "
			}
		} else {
			U.IPs = []string{"N/A"}
		}
		htmlData.Updates = append(htmlData.Updates, U)
	}
	return htmlData
}
