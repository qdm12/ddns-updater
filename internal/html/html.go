package html

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func ConvertRecord(record models.Record, now time.Time) models.HTMLRow {
	const NotAvailable = "N/A"
	row := models.HTMLRow{
		Domain:   convertDomain(record.Settings.BuildDomainName()),
		Host:     models.HTML(record.Settings.Host),
		Provider: convertProvider(record.Settings.Provider),
		IPMethod: convertIPMethod(record.Settings.IPMethod, record.Settings.Provider),
	}
	message := record.Message
	if record.Status == constants.UPTODATE {
		message = "no IP change for " + record.History.GetDurationSinceSuccess(now)
	}
	if len(message) > 0 {
		message = fmt.Sprintf("(%s)", message)
	}
	if len(record.Status) == 0 {
		row.Status = NotAvailable
	} else {
		row.Status = models.HTML(fmt.Sprintf("%s %s, %s",
			convertStatus(record.Status),
			message,
			time.Since(record.Time).Round(time.Second).String()+" ago"))
	}
	currentIP := record.History.GetCurrentIP()
	if currentIP != nil {
		row.CurrentIP = models.HTML(`<a href="https://ipinfo.io/"` + currentIP.String() + `\>` + currentIP.String() + "</a>")
	} else {
		row.CurrentIP = NotAvailable
	}
	previousIPs := record.History.GetPreviousIPs()
	row.PreviousIPs = NotAvailable
	if len(previousIPs) > 0 {
		var previousIPsStr []string
		const maxPreviousIPs = 2
		for i, previousIP := range previousIPs {
			if i == maxPreviousIPs {
				previousIPsStr = append(previousIPsStr, fmt.Sprintf("and %d more", len(previousIPs)-i))
				break
			}
			previousIPsStr = append(previousIPsStr, previousIP.String())
		}
		row.PreviousIPs = models.HTML(strings.Join(previousIPsStr, ", "))
	}
	return row
}

func convertStatus(status models.Status) models.HTML {
	switch status {
	case constants.SUCCESS:
		return constants.HTMLSuccess
	case constants.FAIL:
		return constants.HTMLFail
	case constants.UPTODATE:
		return constants.HTMLUpdate
	case constants.UPDATING:
		return constants.HTMLUpdating
	default:
		return "Unknown status"
	}
}

func convertProvider(provider models.Provider) models.HTML {
	switch provider {
	case constants.NAMECHEAP:
		return constants.HTMLNamecheap
	case constants.GODADDY:
		return constants.HTMLGodaddy
	case constants.DUCKDNS:
		return constants.HTMLDuckDNS
	case constants.DREAMHOST:
		return constants.HTMLDreamhost
	case constants.CLOUDFLARE:
		return constants.HTMLCloudflare
	case constants.NOIP:
		return constants.HTMLNoIP
	case constants.DNSPOD:
		return constants.HTMLDNSPod
	case constants.INFOMANIAK:
		return constants.HTMLInfomaniak
	case constants.DDNSSDE:
		return constants.HTMLDdnssde
	default:
		s := string(provider)
		if strings.HasPrefix("https://", s) {
			shorterName := strings.TrimPrefix(s, "https://")
			shorterName = strings.TrimSuffix(shorterName, "/")
			return models.HTML(fmt.Sprintf("<a href=\"%s\">%s</a>", s, shorterName))
		}
		return models.HTML(string(provider))
	}
}

func convertIPMethod(ipMethod models.IPMethod, provider models.Provider) models.HTML {
	// TODO map to icons
	switch ipMethod {
	case constants.PROVIDER:
		return convertProvider(provider)
	case constants.OPENDNS:
		return constants.HTMLOpenDNS
	case constants.IFCONFIG:
		return constants.HTMLIfconfig
	case constants.IPINFO:
		return constants.HTMLIpinfo
	case constants.IPIFY:
		return constants.HTMLIpify
	case constants.IPIFY6:
		return constants.HTMLIpify6
	case constants.DDNSS:
		return constants.HTMLDdnss
	case constants.DDNSS4:
		return constants.HTMLDdnss4
	case constants.DDNSS6:
		return constants.HTMLDdnss6
	case constants.CYCLE:
		return constants.HTMLCycle
	default:
		return models.HTML(string(ipMethod))
	}
}

func convertDomain(domain string) models.HTML {
	return models.HTML("<a href=\"http://" + domain + "\">" + domain + "</a>")
}
