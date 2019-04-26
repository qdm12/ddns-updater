package models

import "time"

type historyType struct {
	IPs      []string // current and previous ips
	TSuccess time.Time
}

func (history *historyType) string() (s string) {
	if len(history.IPs) > 0 {
		s += "Last success update: " + history.TSuccess.Format("2006-01-02 15:04:05 MST") + "; Current & previous IPs: "
		for i := range history.IPs {
			s += history.IPs[i]
			if i != len(history.IPs)-1 {
				s += ","
			}
		}
	}
	return s
}
