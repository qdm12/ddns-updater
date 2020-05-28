package records

import (
	"fmt"
	"time"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/settings"
)

// Record contains all the information to update and display a DNS record
type Record struct { // internal
	Settings settings.Settings // fixed
	History  models.History    // past information
	Status   models.Status
	Message  string
	Time     time.Time
}

// New returns a new Record with settings and some history
func New(settings settings.Settings, events []models.HistoryEvent) Record {
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
