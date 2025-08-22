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
		message = "No changes needed - IP stable for " + r.History.GetDurationSinceSuccess(now)
	}
	if r.Status == "" {
		row.Status = NotAvailable
	} else {
		timeSince := time.Since(r.Time).Round(time.Second)
		var timeDisplay string
		if timeSince < time.Minute {
			timeDisplay = "Just now"
		} else if timeSince < time.Hour {
			timeDisplay = fmt.Sprintf("%d min ago", int(timeSince.Minutes()))
		} else if timeSince < 24*time.Hour {
			timeDisplay = fmt.Sprintf("%d hrs ago", int(timeSince.Hours()))
		} else {
			timeDisplay = fmt.Sprintf("%d days ago", int(timeSince.Hours()/24))
		}
		
		statusBadge := convertStatus(r.Status)
		if message != "" {
			statusBadge = convertStatusWithTooltip(r.Status, message)
		}
		
		statusHTML := fmt.Sprintf(`%s <span class="status-timestamp">%s</span>`, statusBadge, timeDisplay)
		row.Status = statusHTML
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
		return `<span class="success">Updated</span>`
	case constants.FAIL:
		return `<span class="error">Failed</span>`
	case constants.UPTODATE:
		return `<span class="uptodate">Current</span>`
	case constants.UPDATING:
		return `<span class="updating">Syncing</span>`
	case constants.UNSET:
		return `<span class="unset">Pending</span>`
	default:
		return "Unknown status"
	}
}

func convertStatusWithTooltip(status models.Status, message string) string {
	switch status {
	case constants.SUCCESS:
		return fmt.Sprintf(`<span class="success" title="%s">Updated</span>`, message)
	case constants.FAIL:
		return fmt.Sprintf(`<span class="error" title="%s">Failed</span>`, message)
	case constants.UPTODATE:
		return fmt.Sprintf(`<span class="uptodate" title="%s">Current</span>`, message)
	case constants.UPDATING:
		return fmt.Sprintf(`<span class="updating" title="%s">Syncing</span>`, message)
	case constants.UNSET:
		return fmt.Sprintf(`<span class="unset" title="%s">Pending</span>`, message)
	default:
		return "Unknown status"
	}
}
