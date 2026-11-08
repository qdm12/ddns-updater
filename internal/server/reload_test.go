package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/qdm12/ddns-updater/internal/reload"
	"github.com/stretchr/testify/assert"
)

type stubReloader struct {
	result      reload.Result
	err         error
	period      time.Duration
	periodErr   error
	reloadCalls int
}

func (s *stubReloader) Reload(_ context.Context) (reload.Result, error) {
	s.reloadCalls++
	return s.result, s.err
}

func (s *stubReloader) SetPeriod(period time.Duration) error {
	s.period = period
	return s.periodErr
}

func Test_handlers_reload(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		query          string
		reloader       *stubReloader
		expectedStatus int
		expectedBody   string
		expectedPeriod time.Duration
	}{
		"success": {
			reloader: &stubReloader{result: reload.Result{
				RecordCount: 3,
				Period:      30 * time.Second,
				PeriodSet:   true,
				Cooldown:    time.Minute,
				CooldownSet: true,
			}},
			expectedStatus: http.StatusOK,
			expectedBody:   "Reloaded 3 records; period is 30s; cooldown is 1m0s",
		},
		"success_without_settings": {
			reloader:       &stubReloader{result: reload.Result{RecordCount: 1}},
			expectedStatus: http.StatusOK,
			expectedBody:   "Reloaded 1 records",
		},
		"reload_error": {
			reloader:       &stubReloader{err: assert.AnError},
			expectedStatus: http.StatusInternalServerError,
			expectedBody:   "failed to reload configuration",
		},
		"invalid_period": {
			query:          "?period=abc",
			reloader:       &stubReloader{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "invalid period abc",
		},
		"zero_period": {
			query:          "?period=0s",
			reloader:       &stubReloader{},
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "it must be positive",
		},
		"period_set": {
			query:          "?period=45s",
			reloader:       &stubReloader{result: reload.Result{RecordCount: 2}},
			expectedStatus: http.StatusOK,
			expectedBody:   "Reloaded 2 records",
			expectedPeriod: 45 * time.Second,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			handlers := &handlers{
				ctx:      context.Background(),
				reloader: testCase.reloader,
			}

			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/reload"+testCase.query, nil)
			handlers.reload(recorder, request)

			assert.Equal(t, testCase.expectedStatus, recorder.Code)
			assert.Contains(t, recorder.Body.String(), testCase.expectedBody)
			assert.Equal(t, testCase.expectedPeriod, testCase.reloader.period)
		})
	}
}
