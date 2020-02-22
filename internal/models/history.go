package models

import (
	"fmt"
	"net"
	"strings"
	"time"
)

// History contains current and previous IP address for a particular record
// with the latest success time
type History struct {
	IPs         []net.IP // current and previous ips
	SuccessTime time.Time
}

// GetPreviousIPs returns an antichronological list of previous
// IP addresses if there is any.
func (h *History) GetPreviousIPs() []net.IP {
	if len(h.IPs) <= 1 {
		return nil
	}
	IPs := make([]net.IP, len(h.IPs)-1)
	for i := len(h.IPs) - 2; i >= 0; i-- {
		IPs[i] = h.IPs[i]
	}
	return IPs
}

// GetCurrentIP returns the current IP address (latest in history)
func (h *History) GetCurrentIP() net.IP {
	if len(h.IPs) < 1 {
		return nil
	}
	return h.IPs[len(h.IPs)-1]
}

func (h *History) GetDurationSinceSuccess() string {
	duration := time.Since(h.SuccessTime)
	switch {
	case duration < time.Minute:
		return fmt.Sprintf("%ds", int(duration.Round(time.Second).Seconds()))
	case duration < time.Hour:
		return fmt.Sprintf("%dm", int(duration.Round(time.Minute).Minutes()))
	case duration < 24*time.Hour:
		return fmt.Sprintf("%dh", int(duration.Round(time.Hour).Hours()))
	default:
		return fmt.Sprintf("%dd", int(duration.Round(time.Hour*24).Hours()/24))
	}
}

func (h *History) String() (s string) {
	currentIP := h.GetCurrentIP()
	if currentIP == nil {
		return ""
	}
	successTime := h.SuccessTime
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
