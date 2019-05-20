package models

import (
	"fmt"
	"sync"
	"time"
)

type historyType struct {
	ips      []string // current and previous ips
	tSuccess time.Time
	sync.RWMutex
}

func newHistory(ips []string, tSuccess time.Time) historyType {
	return historyType{
		ips:      ips,
		tSuccess: tSuccess,
	}
}

func (history *historyType) PrependIP(ip string) {
	history.Lock()
	defer history.Unlock()
	history.ips = append([]string{ip}, history.ips...)
}

func (history *historyType) SetTSuccess(t time.Time) {
	history.Lock()
	defer history.Unlock()
	history.tSuccess = t
}

func (history *historyType) GetIPs() []string {
	history.RLock()
	defer history.RUnlock()
	return history.ips
}

func (history *historyType) GetTSuccessDuration() string {
	history.RLock()
	defer history.RUnlock()
	return durationString(history.tSuccess)
}

func durationString(t time.Time) string {
	duration := time.Since(t)
	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Round(time.Second).Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Round(time.Minute).Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Round(time.Hour).Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Round(time.Hour*24).Hours()/24))
	}
}

func (history *historyType) String() (s string) {
	history.RLock()
	defer history.RUnlock()
	if len(history.ips) > 0 {
		s += "Last success update: " + history.tSuccess.Format("2006-01-02 15:04:05 MST") + "; Current and previous IPs: "
		for i := range history.ips {
			s += history.ips[i]
			if i != len(history.ips)-1 {
				s += ","
			}
		}
	}
	return s
}
