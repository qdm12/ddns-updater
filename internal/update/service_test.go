package update

import (
	"context"
	"errors"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/healthchecksio"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/provider"
	librecords "github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
)

type memoryDatabase struct {
	records []librecords.Record
	updates []librecords.Record // every record written, in order
}

func (m *memoryDatabase) Select(id uint) (librecords.Record, error) {
	return m.records[id], nil
}

func (m *memoryDatabase) SelectAll() []librecords.Record {
	return m.records
}

func (m *memoryDatabase) Update(id uint, record librecords.Record) error {
	m.records[id] = record
	m.updates = append(m.updates, record)
	return nil
}

// ipv4Provider is a proxied provider so no DNS lookup is needed.
type ipv4Provider struct {
	provider.Provider
}

func (ipv4Provider) BuildDomainName() string        { return "example.com" }
func (ipv4Provider) Proxied() bool                  { return true }
func (ipv4Provider) IPVersion() ipversion.IPVersion { return ipversion.IP4 }

type ipv4Fetcher struct {
	PublicIPFetcher
	ip netip.Addr // invalid means the IP cannot be found
}

func (f *ipv4Fetcher) IP4(context.Context) (netip.Addr, error) {
	if !f.ip.IsValid() {
		return netip.Addr{}, errors.New("no public IP")
	}
	return f.ip, nil
}

type noopLogger struct{}

func (noopLogger) Debug(string) {}
func (noopLogger) Info(string)  {}
func (noopLogger) Warn(string)  {}
func (noopLogger) Error(string) {}

type noopHealthchecksIO struct{}

func (noopHealthchecksIO) Ping(context.Context, healthchecksio.State) error { return nil }

// A record which did not get a public IP address on the first cycle must
// recover once the IP is found, even if no update is needed then.
func Test_Service_updateNecessary_publicIPFoundLater(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 10, 30, 3, 25, 34, 0, time.UTC)
	currentIP := netip.MustParseAddr("1.2.3.4")
	const period = 10 * time.Minute

	db := &memoryDatabase{records: []librecords.Record{{
		Provider: ipv4Provider{},
		History:  models.History{{IP: currentIP, Time: start.Add(-time.Hour)}},
		Status:   constants.UNSET,
	}}}
	fetcher := &ipv4Fetcher{}
	now := start
	// The updater is left nil since the current IP needs no update.
	service := NewService(db, nil, fetcher, period, 0, noopLogger{}, nil,
		func() time.Time { return now }, noopHealthchecksIO{})

	errs := service.updateNecessary(t.Context())
	assert.Len(t, errs, 1) // public IP fetch error
	record := db.records[0]
	assert.Equal(t, models.Status("no IP"), record.Status)
	assert.Equal(t, "public IP address not found", record.Message)
	assert.Equal(t, start, record.Time)

	now = now.Add(period)
	service.updateNecessary(t.Context())
	record = db.records[0]
	assert.Equal(t, models.Status("no IP"), record.Status)
	assert.Equal(t, start, record.Time, "time must stay at the state change")

	now = now.Add(period)
	fetcher.ip = currentIP
	errs = service.updateNecessary(t.Context())
	assert.Empty(t, errs)
	record = db.records[0]
	assert.Equal(t, constants.UPTODATE, record.Status)
	assert.Empty(t, record.Message)
	assert.Equal(t, now, record.Time)
	assert.Len(t, record.History, 1)
}

func Test_Service_updateNecessary_keepsUpdateFailure(t *testing.T) {
	t.Parallel()

	failTime := time.Date(2026, 10, 30, 3, 25, 34, 0, time.UTC)
	currentIP := netip.MustParseAddr("1.2.3.4")

	db := &memoryDatabase{records: []librecords.Record{{
		Provider: ipv4Provider{},
		History:  models.History{{IP: currentIP, Time: failTime.Add(-time.Hour)}},
		Status:   constants.FAIL,
		Message:  "http 500",
		Time:     failTime,
	}}}
	fetcher := &ipv4Fetcher{ip: currentIP}
	service := NewService(db, nil, fetcher, time.Minute, 0, noopLogger{}, nil,
		func() time.Time { return failTime.Add(time.Minute) }, noopHealthchecksIO{})

	errs := service.updateNecessary(t.Context())

	assert.Empty(t, errs)
	assert.Empty(t, db.updates)
	assert.Equal(t, constants.FAIL, db.records[0].Status)
}
