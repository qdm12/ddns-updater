package models

import "time"

// JSONData is the root structure for the JSON API response
type JSONData struct {
	Records         []JSONRecord `json:"records"`
	Time            time.Time    `json:"time"`
	LastSuccessTime time.Time    `json:"last_success_time"`
	LastSuccessIP   string       `json:"last_success_ip"`
}

// JSONRecord contains all the information for a DNS record in JSON format
type JSONRecord struct {
	Domain            string    `json:"domain"`
	Owner             string    `json:"owner"`
	Provider          string    `json:"provider"`
	IPVersion         string    `json:"ip_version"`
	Status            string    `json:"status"`
	Message           string    `json:"message"`
	CurrentIP         string    `json:"current_ip"`
	PreviousIPs       []string  `json:"previous_ips"`
	TotalIPsInHistory int       `json:"total_ips_in_history"`
	LastUpdate        time.Time `json:"last_update"`
	SuccessTime       time.Time `json:"success_time"`
	Duration          string    `json:"duration_since_success"`
}
