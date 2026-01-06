package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/pkg/publicip/ipversion"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHandlers_JSON(t *testing.T) {
	t.Parallel()

	// Create a mock database with test data
	testRecords := []records.Record{
		{
			Provider: &mockProvider{},
			History: models.History{
				{
					IP:   netip.MustParseAddr("192.168.1.1"),
					Time: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				{
					IP:   netip.MustParseAddr("192.168.1.2"),
					Time: time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC),
				},
				{
					IP:   netip.MustParseAddr("192.168.1.3"),
					Time: time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			Status:  constants.SUCCESS,
			Message: "test message",
			Time:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	mockDB := &mockDatabase{
		records: testRecords,
	}

	// Create handlers
	h := &handlers{
		db:      mockDB,
		timeNow: func() time.Time { return time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC) },
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	w := httptest.NewRecorder()

	// Call the handler
	h.json(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", w.Header().Get("Cache-Control"))

	// Parse response body
	var response models.JSONData
	err := json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)

	// Verify response structure
	assert.Len(t, response.Records, 1)
	assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), response.Time)
	// LastSuccessTime should be from the most recent history entry (History[2] at 12:00:00)
	assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), response.LastSuccessTime)
	assert.Equal(t, "192.168.1.3", response.LastSuccessIP)

	record := response.Records[0]
	assert.Equal(t, "test.example.com", record.Domain)
	assert.Equal(t, "test", record.Owner)
	assert.Equal(t, "mock", record.Provider)
	assert.Equal(t, "ipv4", record.IPVersion)
	assert.Equal(t, "success", record.Status)
	assert.Equal(t, "test message", record.Message)
	assert.Equal(t, "192.168.1.3", record.CurrentIP)
	assert.Len(t, record.PreviousIPs, 2)
	assert.Equal(t, "192.168.1.2", record.PreviousIPs[0]) // most recent previous
	assert.Equal(t, "192.168.1.1", record.PreviousIPs[1]) // oldest previous
	assert.Equal(t, 2, record.TotalIPsInHistory)
}

func TestHandlers_JSON_EmptyRecords(t *testing.T) {
	t.Parallel()

	// Create a mock database with no records
	mockDB := &mockDatabase{
		records: []records.Record{},
	}

	// Create handlers
	h := &handlers{
		db:      mockDB,
		timeNow: func() time.Time { return time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC) },
	}

	// Create request
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	w := httptest.NewRecorder()

	// Call the handler
	h.json(w, req)

	// Check response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/json", w.Header().Get("Content-Type"))

	// Get the body string before decoding (since decoding consumes the body)
	bodyBytes := w.Body.Bytes()
	bodyStr := string(bodyBytes)

	// Verify the raw JSON contains "[]" (with possible whitespace) and not "null" for records
	assert.Regexp(t, `"records"\s*:\s*\[\]`, bodyStr)
	assert.NotContains(t, bodyStr, `"records":null`)

	// Parse response body
	var response models.JSONData
	err := json.Unmarshal(bodyBytes, &response)
	require.NoError(t, err)

	// Verify response structure - records should be an empty array, not null
	assert.NotNil(t, response.Records)
	assert.Len(t, response.Records, 0)
	assert.Equal(t, time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC), response.Time)
	assert.True(t, response.LastSuccessTime.IsZero())
	assert.Empty(t, response.LastSuccessIP)
}

// Mock database for testing
type mockDatabase struct {
	records []records.Record
}

func (m *mockDatabase) SelectAll() []records.Record {
	return m.records
}

func (m *mockDatabase) Start(ctx context.Context) (<-chan error, error) {
	return nil, nil
}

func (m *mockDatabase) Stop() error {
	return nil
}

// Mock provider for testing
type mockProvider struct{}

func (m *mockProvider) String() string {
	return "[domain: test.example.com | owner: test | provider: mock | ip: ipv4]"
}

func (m *mockProvider) Name() models.Provider {
	return "mock"
}

func (m *mockProvider) Domain() string {
	return "test.example.com"
}

func (m *mockProvider) Owner() string {
	return "test"
}

func (m *mockProvider) BuildDomainName() string {
	return "test.test.example.com"
}

func (m *mockProvider) HTML() models.HTMLRow {
	return models.HTMLRow{
		Domain:      "test.example.com",
		Owner:       "test",
		Provider:    "mock",
		IPVersion:   "ipv4",
		Status:      "",
		CurrentIP:   "",
		PreviousIPs: "",
	}
}

func (m *mockProvider) Proxied() bool {
	return false
}

func (m *mockProvider) IPVersion() ipversion.IPVersion {
	return ipversion.IP4
}

func (m *mockProvider) IPv6Suffix() netip.Prefix {
	return netip.Prefix{}
}

func (m *mockProvider) Update(ctx context.Context, client *http.Client, ip netip.Addr) (newIP netip.Addr, err error) {
	return ip, nil
}
