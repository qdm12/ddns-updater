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

	// Format IP version with icon/badge
	row.IPVersion = formatIPVersion(row.IPVersion)

	// Set status class for row background tinting
	row.StatusClass = getStatusClass(r.Status)

	// Build status display
	if r.Status == "" {
		row.Status = NotAvailable
	} else {
		// Build message for tooltip
		var fullMessage string
		if r.Status == constants.UPTODATE {
			duration := r.History.GetDurationSinceSuccess(now)
			fullMessage = fmt.Sprintf("No IP change for %s", duration)
		} else if r.Message != "" {
			fullMessage = r.Message
		}

		// Create status badge with tooltip
		statusBadge := convertStatusWithTooltip(r.Status, fullMessage)
		lastUpdate := formatTimeSince(r.Time, now)

		// Combine badge and time inline only
		row.Status = fmt.Sprintf(`<div class="status-container"><div class="status-inline">%s<span class="status-time">%s</span></div></div>`,
			statusBadge,
			lastUpdate)
	}

	// Format current IP
	currentIP := r.History.GetCurrentIP()
	if currentIP.IsValid() {
		row.CurrentIP = `<a href="https://ipinfo.io/` + currentIP.String() + `">` + currentIP.String() + "</a>"
	} else {
		row.CurrentIP = NotAvailable
	}

	// Format previous IPs
	previousIPs := r.History.GetPreviousIPs()
	row.PreviousIPs = NotAvailable
	if len(previousIPs) > 0 {
		var previousIPsHTML []string
		const maxPreviousIPs = 2
		for i, previousIP := range previousIPs {
			if i == maxPreviousIPs {
				previousIPsHTML = append(previousIPsHTML,
					fmt.Sprintf(`<span class="text-muted">+%d more</span>`, len(previousIPs)-i))
				break
			}
			previousIPsHTML = append(previousIPsHTML,
				fmt.Sprintf(`<span class="ip-badge">%s</span>`, previousIP.String()))
		}
		row.PreviousIPs = strings.Join(previousIPsHTML, " ")
	}
	return row
}

// formatTimeSince formats a duration in a human-readable way
func formatTimeSince(t time.Time, now time.Time) string {
	duration := now.Sub(t)

	// Just now (less than 10 seconds)
	if duration < 10*time.Second {
		return "just now"
	}

	// Seconds (less than 1 minute)
	if duration < time.Minute {
		seconds := int(duration.Seconds())
		return fmt.Sprintf("%ds ago", seconds)
	}

	// Minutes (less than 1 hour)
	if duration < time.Hour {
		minutes := int(duration.Minutes())
		return fmt.Sprintf("%dm ago", minutes)
	}

	// Hours (less than 1 day)
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		if minutes > 0 {
			return fmt.Sprintf("%dh %dm ago", hours, minutes)
		}
		return fmt.Sprintf("%dh ago", hours)
	}

	// Days (less than 7 days)
	if duration < 7*24*time.Hour {
		days := int(duration.Hours() / 24)
		hours := int(duration.Hours()) % 24
		if hours > 0 {
			return fmt.Sprintf("%dd %dh ago", days, hours)
		}
		return fmt.Sprintf("%dd ago", days)
	}

	// Weeks
	weeks := int(duration.Hours() / 24 / 7)
	days := int(duration.Hours()/24) % 7
	if days > 0 {
		return fmt.Sprintf("%dw %dd ago", weeks, days)
	}
	return fmt.Sprintf("%dw ago", weeks)
}

func convertStatusWithTooltip(status models.Status, message string) string {
	// Determine if we need tooltip class
	tooltipClass := ""
	tooltipAttr := ""
	if message != "" {
		tooltipClass = " has-status-tooltip"
		tooltipAttr = fmt.Sprintf(` data-tooltip="%s"`, escapeHTML(message))
	}

	switch status {
	case constants.SUCCESS:
		return fmt.Sprintf(`<span class="badge badge-success%s"%s>`, tooltipClass, tooltipAttr) +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<polyline points="20 6 9 17 4 12"></polyline>` +
			`</svg>` +
			`Success</span>`
	case constants.FAIL:
		return fmt.Sprintf(`<span class="badge badge-error%s"%s>`, tooltipClass, tooltipAttr) +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<circle cx="12" cy="12" r="10"></circle>` +
			`<line x1="15" y1="9" x2="9" y2="15"></line>` +
			`<line x1="9" y1="9" x2="15" y2="15"></line>` +
			`</svg>` +
			`Failed</span>`
	case constants.UPTODATE:
		return fmt.Sprintf(`<span class="badge badge-success%s"%s>`, tooltipClass, tooltipAttr) +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<polyline points="20 6 9 17 4 12"></polyline>` +
			`</svg>` +
			`Up to date</span>`
	case constants.UPDATING:
		return fmt.Sprintf(`<span class="badge badge-info%s"%s>`, tooltipClass, tooltipAttr) +
			`<svg class="animate-spin" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<path d="M21 12a9 9 0 11-6.219-8.56"></path>` +
			`</svg>` +
			`Updating</span>`
	case constants.UNSET:
		return fmt.Sprintf(`<span class="badge badge-warning%s"%s>`, tooltipClass, tooltipAttr) +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<path d="M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"></path>` +
			`<line x1="12" y1="9" x2="12" y2="13"></line>` +
			`<line x1="12" y1="17" x2="12.01" y2="17"></line>` +
			`</svg>` +
			`Unset</span>`
	default:
		return "Unknown status"
	}
}

// formatIPVersion formats IP version string with styled badge and icon
func formatIPVersion(ipVersion string) string {
	switch strings.ToLower(ipVersion) {
	case "ipv4":
		return `<span class="ip-version-badge ipv4-badge">` +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<rect x="2" y="2" width="20" height="8" rx="2" ry="2"></rect>` +
			`<rect x="2" y="14" width="20" height="8" rx="2" ry="2"></rect>` +
			`<line x1="6" y1="6" x2="6.01" y2="6"></line>` +
			`<line x1="6" y1="18" x2="6.01" y2="18"></line>` +
			`</svg>` +
			`IPv4</span>`
	case "ipv6":
		return `<span class="ip-version-badge ipv6-badge">` +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<polygon points="12 2 2 7 12 12 22 7 12 2"></polygon>` +
			`<polyline points="2 17 12 22 22 17"></polyline>` +
			`<polyline points="2 12 12 17 22 12"></polyline>` +
			`</svg>` +
			`IPv6</span>`
	case "ipv4 or ipv6":
		return `<span class="ip-version-badge ipv4v6-badge">` +
			`<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` +
			`<circle cx="12" cy="12" r="10"></circle>` +
			`<line x1="2" y1="12" x2="22" y2="12"></line>` +
			`<path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path>` +
			`</svg>` +
			`IPv4/6</span>`
	default:
		return ipVersion
	}
}

// escapeHTML escapes special HTML characters to prevent XSS
func escapeHTML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&#39;")
	return s
}

// getStatusClass returns CSS class for row background tinting
func getStatusClass(status models.Status) string {
	switch status {
	case constants.SUCCESS, constants.UPTODATE:
		return "status-success"
	case constants.FAIL:
		return "status-error"
	case constants.UPDATING:
		return "status-updating"
	case constants.UNSET:
		return "status-warning"
	default:
		return ""
	}
}
