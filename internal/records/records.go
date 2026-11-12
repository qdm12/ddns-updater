package records

import (
	"fmt"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
)

// Record contains all the information to update and display a DNS record.
// Only the provider and the history are persisted to disk. The status, message,
// time and last ban fields live in memory only and are reset when the program
// restarts, which is why a restart clears a stale status on its own.
type Record struct { // internal
	Provider provider.Provider // fixed
	History  models.History    // past information, persisted
	Status   models.Status     // in memory only
	Message  string            // in memory only, describes Status
	Time     time.Time         // in memory only, time of Status
	LastBan  *time.Time        // in memory only, nil means no last ban
}

// New returns a new Record with provider and some history.
func New(provider provider.Provider, events []models.HistoryEvent) Record {
	return Record{
		Provider: provider,
		History:  events,
		Status:   constants.UNSET,
	}
}

func (r *Record) String() string {
	status := string(r.Status)
	if r.Message != "" {
		status += " (" + r.Message + ")"
	}
	return fmt.Sprintf("%s: %s %s; %s",
		r.Provider, status, r.Time.Format("2006-01-02 15:04:05 MST"), r.History)
}
