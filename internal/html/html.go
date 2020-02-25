package html

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func ConvertRecord(record models.Record, now time.Time) models.HTMLRow {
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
		row.Status = "N/A"
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
		row.CurrentIP = "N/A"
	}
	previousIPs := record.History.GetPreviousIPs()
	row.PreviousIPs = "N/A"
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
		return constants.HTML_SUCCESS
	case constants.FAIL:
		return constants.HTML_FAIL
	case constants.UPTODATE:
		return constants.HTML_UPTODATE
	case constants.UPDATING:
		return constants.HTML_UPDATING
	default:
		return "Unknown status"
	}
}

func convertProvider(provider models.Provider) models.HTML {
	switch provider {
	case constants.NAMECHEAP:
		return constants.HTML_NAMECHEAP
	case constants.GODADDY:
		return constants.HTML_GODADDY
	case constants.DUCKDNS:
		return constants.HTML_DUCKDNS
	case constants.DREAMHOST:
		return constants.HTML_DREAMHOST
	case constants.CLOUDFLARE:
		return constants.HTML_CLOUDFLARE
	case constants.NOIP:
		return constants.HTML_NOIP
	case constants.DNSPOD:
		return constants.HTML_DNSPOD
	case constants.INFOMANIAK:
		return constants.HTML_INFOMANIAK
	default:
		return models.HTML(string(provider))
	}
}

func convertIPMethod(IPMethod models.IPMethod, provider models.Provider) models.HTML {
	// TODO map to icons
	switch IPMethod {
	case constants.PROVIDER:
		return convertProvider(provider)
	case constants.GOOGLE:
		return constants.HTML_GOOGLE
	case constants.OPENDNS:
		return constants.HTML_OPENDNS
	default:
		return models.HTML(string(IPMethod))
	}
}

func convertDomain(domain string) models.HTML {
	return models.HTML("<a href=\"http://" + domain + "\">" + domain + "</a>")
}
