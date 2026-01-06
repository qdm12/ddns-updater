package records

import (
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func (r *Record) JSON(now time.Time) models.JSONRecord {
	const NotAvailable = "N/A"

	// Determine status and message
	status := string(r.Status)
	message := r.Message
	if r.Status == constants.UPTODATE {
		message = "no IP change for " + r.History.GetDurationSinceSuccess(now)
	}

	// Get current IP
	currentIP := r.History.GetCurrentIP()
	currentIPStr := NotAvailable
	if currentIP.IsValid() {
		currentIPStr = currentIP.String()
	}

	// Get previous IPs (limit to last 10, similar to HTML view)
	previousIPs := r.History.GetPreviousIPs()
	var previousIPsStr []string
	totalIPsInHistory := len(previousIPs)

	if len(previousIPs) > 0 {
		const maxPreviousIPs = 10
		for i, ip := range previousIPs {
			if i == maxPreviousIPs {
				break
			}
			previousIPsStr = append(previousIPsStr, ip.String())
		}
	}

	return models.JSONRecord{
		Domain:            r.Provider.Domain(),
		Owner:             r.Provider.Owner(),
		Provider:          string(r.Provider.Name()),
		IPVersion:         r.Provider.IPVersion().String(),
		Status:            status,
		Message:           message,
		CurrentIP:         currentIPStr,
		PreviousIPs:       previousIPsStr,
		TotalIPsInHistory: totalIPsInHistory,
		LastUpdate:        r.Time,
		SuccessTime:       r.History.GetSuccessTime(),
		Duration:          r.History.GetDurationSinceSuccess(now),
	}
}
