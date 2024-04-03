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
	row := r.Provider.HTML()
	message := r.Message
	if r.Status == constants.UPTODATE {
		message = "no IP change for " + r.History.GetDurationSinceSuccess(now)
	}
	if message != "" {
		message = fmt.Sprintf("(%s)", message)
	}
	if r.Status == "" {
		row.Status = NotAvailable
	} else {
		row.Status = fmt.Sprintf("%s %s, %s",
			convertStatus(r.Status),
			message,
			time.Since(r.Time).Round(time.Second).String()+" ago")
	}
	currentIP := r.History.GetCurrentIP()
	if currentIP.IsValid() {
		row.CurrentIP = `<a href="https://ipinfo.io/` + currentIP.String() + `">` + currentIP.String() + "</a>"
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
		row.PreviousIPs = strings.Join(previousIPsStr, ", ")
	}
	return row
}

func convertStatus(status models.Status) string {
	switch status {
	case constants.SUCCESS:
		return `<span class="success">Success</span>`
	case constants.FAIL:
		return `<span class="error">Failure</span>`
	case constants.UPTODATE:
		return `<span class="uptodate">Up to date</span>`
	case constants.UPDATING:
		return `<span class="updating">Updating</span>`
	case constants.UNSET:
		return `<span class="unset">Unset</span>`
	default:
		return "Unknown status"
	}
}
