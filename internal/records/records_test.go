package records

import (
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_Record_IsFailing(t *testing.T) {
	t.Parallel()

	testCases := map[models.Status]bool{
		constants.FAIL:     true,
		constants.NOIP:     true,
		constants.SUCCESS:  false,
		constants.UPTODATE: false,
		constants.UPDATING: false,
		constants.UNSET:    false,
	}

	for status, failing := range testCases {
		t.Run(string(status), func(t *testing.T) {
			t.Parallel()
			record := Record{Status: status}
			assert.Equal(t, failing, record.IsFailing())
		})
	}
}

func Test_convertStatus_noIP(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `<span class="error">No IP</span>`, convertStatus(constants.NOIP))
}

func Test_Record_SetStatus(t *testing.T) {
	t.Parallel()

	first := time.Date(2026, 10, 30, 3, 25, 34, 0, time.UTC)
	second := first.Add(time.Minute)
	third := second.Add(time.Minute)

	record := Record{Status: constants.FAIL, Message: "http 500"}

	record.SetStatus(constants.NOIP, "not found", first)
	assert.Equal(t, Record{Status: constants.NOIP, Message: "not found", Time: first}, record)

	record.SetStatus(constants.NOIP, "not found", second)
	assert.Equal(t, first, record.Time, "time must not change for the same state")

	record.SetStatus(constants.NOIP, "other", second)
	assert.Equal(t, Record{Status: constants.NOIP, Message: "other", Time: second}, record)

	record.SetStatus(constants.UPTODATE, "", third)
	assert.Equal(t, Record{Status: constants.UPTODATE, Time: third}, record)
}
