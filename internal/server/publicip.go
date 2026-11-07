package server

import (
	"context"
	"net/http"
	"time"

	"github.com/qdm12/ddns-updater/pkg/publicip"
)

type lookupJSON struct {
	IP         string `json:"ip,omitempty"`
	Error      string `json:"error,omitempty"`
	DurationMs int64  `json:"durationMs"`
}

type publicIPResultJSON struct {
	Method string     `json:"method"`
	IPv4   lookupJSON `json:"ipv4"`
	IPv6   lookupJSON `json:"ipv6"`
}

type publicIPResponseJSON struct {
	CheckedAt time.Time            `json:"checkedAt"`
	Results   []publicIPResultJSON `json:"results"`
}

func toLookupJSON(result publicip.FetchResult) lookupJSON {
	lookup := lookupJSON{DurationMs: result.Duration.Milliseconds()}
	if result.Err != nil {
		lookup.Error = result.Err.Error()
	} else {
		lookup.IP = result.IP.String()
	}
	return lookup
}

func (h *handlers) publicIPPage(w http.ResponseWriter, _ *http.Request) {
	data := struct{ Settings string }{Settings: h.publicIPSettings}
	renderPage(w, h.publicIPTemplate, "publicip.html", data)
}

// publicIPCheck runs a public IP lookup with each enabled method
// and returns the results as JSON.
func (h *handlers) publicIPCheck(w http.ResponseWriter, r *http.Request) {
	if rejectCrossSite(w, r) {
		return
	}

	if !h.publicIPMutex.TryLock() {
		writeJSONError(w, http.StatusTooManyRequests, "a public IP check is already running")
		return
	}
	defer h.publicIPMutex.Unlock()

	const checkTimeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(r.Context(), checkTimeout)
	defer cancel()

	results := h.publicIPChecker.Check(ctx)

	response := publicIPResponseJSON{
		CheckedAt: h.timeNow(),
		Results:   make([]publicIPResultJSON, len(results)),
	}
	for i, result := range results {
		response.Results[i] = publicIPResultJSON{
			Method: result.Method,
			IPv4:   toLookupJSON(result.IPv4),
			IPv6:   toLookupJSON(result.IPv6),
		}
	}
	writeJSON(w, http.StatusOK, response)
}
