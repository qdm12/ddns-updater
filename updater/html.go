package main

import "time"

type HtmlData struct {
	Updates []UpdateType
}

type UpdateType struct { // exported for HTML template
	Domain   string
	Host     string
	Provider string
	IpMethod string
	Status   string
	Duration string
	IP       string   // current set ip
	IPs      []string // previous ips
}

func updatesToHtml(updates *Updates) (htmlData *HtmlData) {
	htmlData = new(HtmlData)
	for _, u := range *updates {
		var U UpdateType
		U.Domain = u.settings.htmlDomain()
		U.Host = u.settings.host // TODO html method
		U.Provider = u.settings.htmlProvider()
		U.IpMethod = u.settings.htmlIpmethod()
		if u.status.code == UPTODATE {
			u.status.message = "No IP change for " + time.Since(u.extras.tSuccess).Round(time.Second).String()
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
