package records

import (
	"fmt"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
)

// Record contains all the information to update and display a DNS record.
type Record struct { // internal
	Provider provider.Provider // fixed
	History  models.History    // past information
	Status   models.Status
	Message  string
	Time     time.Time
	LastBan  *time.Time // nil means no last ban
}

// New returns a new Record with provider and some history.
func New(provider provider.Provider, events []models.HistoryEvent) Record {
	return Record{
		Provider: provider,
		History:  events,
		Status:   constants.UNSET,
	}
}

// SetStatus records a state change. Time is when the state last changed,
// so setting the same status and message again changes nothing.
func (r *Record) SetStatus(status models.Status, message string, now time.Time) {
	if r.Status == status && r.Message == message {
		return
	}
	r.Status, r.Message, r.Time = status, message, now
}

// IsFailing returns true if the record has no working update to rely on,
// either because an update failed or its public IP could not be found.
func (r *Record) IsFailing() bool {
	return r.Status == constants.FAIL || r.Status == constants.NOIP
}

func (r *Record) String() string {
	status := string(r.Status)
	if r.Message != "" {
		status += " (" + r.Message + ")"
	}
	return fmt.Sprintf("%s: %s %s; %s",
		r.Provider, status, r.Time.Format("2006-01-02 15:04:05 MST"), r.History)
}
