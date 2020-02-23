package models

import (
	"fmt"
	"time"
)

// Record contains all the information to update and display a DNS record
type Record struct { // internal
	Settings Settings // fixed
	History  History  // past information
	Status   Status
	Message  string
	Time     time.Time
}

// NewRecord returns a new Record with settings and some history
func NewRecord(settings Settings, events []HistoryEvent) Record {
	return Record{
		Settings: settings,
		History:  events,
	}
}

func (r *Record) String() string {
	status := string(r.Status)
	if len(r.Message) > 0 {
		status += " (" + r.Message + ")"
	}
	return fmt.Sprintf("%s: %s %s; %s", r.Settings.String(), status, r.Time.Format("2006-01-02 15:04:05 MST"), r.History.String())
}
