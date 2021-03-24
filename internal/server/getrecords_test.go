package server

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/data/mock_data"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/ddns-updater/internal/records"
	"github.com/qdm12/ddns-updater/internal/settings/mock_settings"
	"github.com/stretchr/testify/assert"
)

func Test_handlers_getRecords(t *testing.T) {
	t.Parallel()

	exampleTime := time.Unix(1000, 0)

	exampleRecord := records.Record{
		History: models.History{
			models.HistoryEvent{
				IP:   net.IP{127, 0, 0, 1},
				Time: exampleTime,
			},
		},
		Status:  constants.SUCCESS,
		Message: "message",
		Time:    exampleTime,
		LastBan: &exampleTime,
	}

	testCases := map[string]struct {
		records      []records.Record
		responseBody string
	}{
		"empty records": {
			records:      []records.Record{},
			responseBody: "[]\n",
		},
		"single record": {
			records: []records.Record{
				exampleRecord,
			},
			responseBody: `[{"Settings":{"a":{}},"History":[{"ip":"127.0.0.1","time":"1970-01-01T00:16:40Z"}],"Status":"success","Message":"message","Time":"1970-01-01T00:16:40Z","LastBan":"1970-01-01T00:16:40Z"}]` + "\n", // nolint:lll
		},
	}

	for name, testCase := range testCases {
		testCase := testCase
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)

			// Set the settings interface on each record
			for i := range testCase.records {
				settings := mock_settings.NewMockSettings(ctrl)
				const json = `{"a":{}}`
				settings.EXPECT().MarshalJSON().Return([]byte(json), nil)
				testCase.records[i].Settings = settings
			}

			db := mock_data.NewMockDatabase(ctrl)
			db.EXPECT().SelectAll().Return(testCase.records)

			handlers := &handlers{
				db: db,
			}

			w := httptest.NewRecorder()

			handlers.getRecords(w, nil)

			response := w.Result()
			assert.Equal(t, http.StatusOK, response.StatusCode)
			assert.Equal(t, testCase.responseBody, w.Body.String())
			_ = response.Body.Close()
		})
	}
}
