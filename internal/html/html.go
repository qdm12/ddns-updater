package html

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func ConvertRecord(record models.Record) models.HTMLRow {
	row := models.HTMLRow{
		Domain:   convertDomain(record.Settings.BuildDomainName()),
		Host:     record.Settings.Host,
		Provider: convertProvider(record.Settings.Provider),
		IPMethod: convertIPMethod(record.Settings.IPMethod, record.Settings.Provider),
	}
	message := record.Message
	if record.Status == constants.UPTODATE {
		message = "no IP change for " + record.History.GetDurationSinceSuccess()
	}
	if len(message) > 0 {
		message = fmt.Sprintf("(%s)", message)
	}
	if len(record.Status) == 0 {
		row.Status = "N/A"
	} else {
		row.Status = fmt.Sprintf("%s %s, %s",
			convertStatus(record.Status),
			message,
			time.Since(record.Time).Round(time.Second).String()+" ago")
	}
	currentIP := record.History.GetCurrentIP()
	if currentIP != nil {
		row.CurrentIP = `<a href="https://ipinfo.io/"` + currentIP.String() + `\>` + currentIP.String() + "</a>"
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
		row.PreviousIPs = strings.Join(previousIPsStr, ", ")
	}
	return row
}

func convertStatus(status models.Status) string {
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

func convertProvider(provider models.Provider) string {
	switch provider {
	case constants.PROVIDERNAMECHEAP:
		return "<a href=\"https://namecheap.com\">Namecheap</a>"
	case constants.PROVIDERGODADDY:
		return "<a href=\"https://godaddy.com\">GoDaddy</a>"
	case constants.PROVIDERDUCKDNS:
		return "<a href=\"https://duckdns.org\">DuckDNS</a>"
	case constants.PROVIDERDREAMHOST:
		return "<a href=\"https://https://www.dreamhost.com/\">Dreamhost</a>"
	default:
		return string(provider)
	}
}

func convertIPMethod(IPMethod models.IPMethod, provider models.Provider) string {
	// TODO map to icons
	switch IPMethod {
	case constants.IPMETHODPROVIDER:
		return convertProvider(provider)
	case constants.IPMETHODGOOGLE:
		return "<a href=\"https://google.com/search?q=ip\">Google</a>"
	case constants.IPMETHODOPENDNS:
		return "<a href=\"https://diagnostic.opendns.com/myip\">OpenDNS</a>"
	default:
		return string(IPMethod)
	}
}

func convertDomain(domain string) string {
	return "<a href=\"http://" + domain + "\">" + domain + "</a>"
}
