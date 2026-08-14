package server

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"time"

	"github.com/qdm12/ddns-updater/internal/provider"
	"github.com/qdm12/ddns-updater/internal/records"
)

// apiRecord is the machine readable counterpart of [models.HTMLRow].
// Fields are built explicitly from the provider interface rather than
// marshaling the provider itself, which carries credentials.
type apiRecord struct {
	Domain      string     `json:"domain"`
	Owner       string     `json:"owner"`
	FQDN        string     `json:"fqdn"`
	Provider    string     `json:"provider"`
	IPVersion   string     `json:"ip_version"`
	Proxied     bool       `json:"proxied"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	CurrentIP   string     `json:"current_ip,omitempty"`
	PreviousIPs []string   `json:"previous_ips"`
	LastAttempt *time.Time `json:"last_attempt"`
	LastSuccess *time.Time `json:"last_success"`
	LastBan     *time.Time `json:"last_ban"`
}

type apiSummary struct {
	Total    int            `json:"total"`
	ByStatus map[string]int `json:"by_status"`
}

type apiRecordsResponse struct {
	Records []apiRecord `json:"records"`
	Summary apiSummary  `json:"summary"`
}

func (h *handlers) apiRecords(w http.ResponseWriter, _ *http.Request) {
	all := h.db.SelectAll()

	response := apiRecordsResponse{
		Records: make([]apiRecord, 0, len(all)),
		Summary: apiSummary{
			Total:    len(all),
			ByStatus: make(map[string]int, len(all)),
		},
	}

	for _, record := range all {
		response.Records = append(response.Records, toAPIRecord(record))
		response.Summary.ByStatus[string(record.Status)]++
	}

	writeJSON(w, http.StatusOK, response)
}

func toAPIRecord(record records.Record) apiRecord {
	providerName := ""
	if namer, ok := record.Provider.(provider.Namer); ok {
		providerName = string(namer.Name())
	}

	previousIPs := record.History.GetPreviousIPs()
	previousIPsStr := make([]string, 0, len(previousIPs))
	for _, previousIP := range previousIPs {
		previousIPsStr = append(previousIPsStr, previousIP.String())
	}

	return apiRecord{
		Domain:      record.Provider.Domain(),
		Owner:       record.Provider.Owner(),
		FQDN:        record.Provider.BuildDomainName(),
		Provider:    providerName,
		IPVersion:   record.Provider.IPVersion().String(),
		Proxied:     record.Provider.Proxied(),
		Status:      string(record.Status),
		Message:     record.Message,
		CurrentIP:   ipString(record.History.GetCurrentIP()),
		PreviousIPs: previousIPsStr,
		LastAttempt: timePtr(record.Time),
		LastSuccess: timePtr(record.History.GetSuccessTime()),
		LastBan:     record.LastBan,
	}
}

func (h *handlers) apiVersion(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, h.buildInfo)
}

func ipString(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	return ip.String()
}

// timePtr returns nil for the zero time so consumers get a JSON null instead
// of a year 1 timestamp, which would read as a real one.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		// The status code and possibly part of the body are already written,
		// so the error can only be logged through the response being truncated.
		return
	}
}
