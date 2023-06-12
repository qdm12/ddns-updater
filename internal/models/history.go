package models

import (
	"fmt"
	"net/netip"
	"strings"
	"time"
)

// History contains current and previous IP address for a particular record
// with the latest success time.
type History []HistoryEvent // current and previous ips

type HistoryEvent struct { // current and previous ips
	IP   netip.Addr `json:"ip"`
	Time time.Time  `json:"time"`
}

// GetPreviousIPs returns an antichronological list of previous
// IP addresses if there is any.
func (h History) GetPreviousIPs() []netip.Addr {
	if len(h) <= 1 {
		return nil
	}
	IPs := make([]netip.Addr, len(h)-1)
	const two = 2
	for i := len(h) - two; i >= 0; i-- {
		IPs[i] = h[i].IP
	}
	return IPs
}

// GetCurrentIP returns the current IP address (latest in history).
func (h History) GetCurrentIP() netip.Addr {
	if len(h) < 1 {
		return netip.Addr{}
	}
	return h[len(h)-1].IP
}

// GetSuccessTime returns the latest success update time.
func (h History) GetSuccessTime() time.Time {
	if len(h) < 1 {
		return time.Time{}
	}
	return h[len(h)-1].Time
}

func (h History) GetDurationSinceSuccess(now time.Time) string {
	if len(h) < 1 {
		return "N/A"
	}
	duration := now.Sub(h[len(h)-1].Time)
	const hoursInDay = 24
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds", int(duration.Round(time.Second).Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm", int(duration.Round(time.Minute).Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh", int(duration.Round(time.Hour).Hours()))
	default:
		return fmt.Sprintf("%dd", int(duration.Round(time.Hour*hoursInDay).Hours()/hoursInDay))
	}
}

func (h History) String() (s string) {
	currentIP := h.GetCurrentIP()
	if !currentIP.IsValid() {
		return ""
	}
	successTime := h[len(h)-1].Time
	previousIPs := h.GetPreviousIPs()
	if len(previousIPs) == 0 {
		return fmt.Sprintf(
			"Last success update: %s; IP: %s",
			successTime.Format("2006-01-02 15:04:05 MST"),
			currentIP.String(),
		)
	}
	const maxDisplay = 4
	previousIPsStr := []string{}
	for i, IP := range previousIPs {
		if i == maxDisplay {
			previousIPsStr = append(previousIPsStr, fmt.Sprintf("...(%d more)", len(previousIPs)-maxDisplay))
			break
		}
		previousIPsStr = append(previousIPsStr, IP.String())
	}
	return fmt.Sprintf(
		"Last success update: %s; IP: %s; Previous IPs: %s",
		successTime.Format("2006-01-02 15:04:05 MST"),
		currentIP.String(),
		strings.Join(previousIPsStr, ","),
	)
}
