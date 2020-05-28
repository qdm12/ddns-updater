package records

import (
	"fmt"
	"strings"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func (r *Record) HTML(now time.Time) models.HTMLRow {
	const NotAvailable = "N/A"
	row := r.Settings.HTML()
	message := r.Message
	if r.Status == constants.UPTODATE {
		message = "no IP change for " + r.History.GetDurationSinceSuccess(now)
	}
	if len(message) > 0 {
		message = fmt.Sprintf("(%s)", message)
	}
	if len(r.Status) == 0 {
		row.Status = NotAvailable
	} else {
		row.Status = models.HTML(fmt.Sprintf("%s %s, %s",
			convertStatus(r.Status),
			message,
			time.Since(r.Time).Round(time.Second).String()+" ago"))
	}
	currentIP := r.History.GetCurrentIP()
	if currentIP != nil {
		row.CurrentIP = models.HTML(`<a href="https://ipinfo.io/"` + currentIP.String() + `\>` + currentIP.String() + "</a>")
	} else {
		row.CurrentIP = NotAvailable
	}
	previousIPs := r.History.GetPreviousIPs()
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
		return `<font color="green"><b>Success</b></font>`
	case constants.FAIL:
		return `<font color="red"><b>Failure</b></font>`
	case constants.UPTODATE:
		return `<font color="#00CC66"><b>Up to date</b></font>`
	case constants.UPDATING:
		return `<font color="orange"><b>Updating</b></font>`
	default:
		return "Unknown status"
	}
}
