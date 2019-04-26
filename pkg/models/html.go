package models

import (
	"fmt"
	"time"
)

// HTMLData is a list of HTML fields to be rendered.
// It is exported so that the HTML template engine can render it.
type HTMLData struct {
	Rows []HTMLRow
}

// HTMLRow contains HTML fields to be rendered
// It is exported so that the HTML template engine can render it.
type HTMLRow struct {
	Domain   string
	Host     string
	Provider string
	IPMethod string
	Status   string
	IP       string   // current set ip
	IPs      []string // previous ips
}

// ToHTML converts all the update record configs to HTML data ready to be templated
func ToHTML(recordsConfigs []RecordConfigType) (htmlData HTMLData) {
	for i := range recordsConfigs {
		htmlData.Rows = append(htmlData.Rows, recordsConfigs[i].toHTML())
	}
	return htmlData
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
