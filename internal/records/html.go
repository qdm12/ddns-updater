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
		row.CurrentIP = `<a href="https://ipinfo.io/` + currentIP.String() + `" class="ip-link" target="_blank" rel="noopener noreferrer">` + currentIP.String() + ` <svg class="ipinfo-icon" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg"><path d="M14 3v2h3.59l-9.83 9.83 1.41 1.41L19 6.41V10h2V3m-2 16H5V5h7V3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2v-7h-2v7z" fill="currentColor"/></svg></a>`
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
		return fmt.Sprintf(`<span class="success" data-tooltip="%s">Updated</span>`, message)
	case constants.FAIL:
		return fmt.Sprintf(`<span class="error" data-tooltip="%s">Failed</span>`, message)
	case constants.UPTODATE:
		return fmt.Sprintf(`<span class="uptodate" data-tooltip="%s">Current</span>`, message)
	case constants.UPDATING:
		return fmt.Sprintf(`<span class="updating" data-tooltip="%s">Syncing</span>`, message)
	case constants.UNSET:
		return fmt.Sprintf(`<span class="unset" data-tooltip="%s">Pending</span>`, message)
	default:
		return "Unknown status"
	}
}
