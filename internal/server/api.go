package server

import (
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/records"
)

// apiRecord is built field by field rather than marshaling the provider,
// which holds credentials.
type apiRecord struct {
	Domain           string     `json:"domain"`
	Owner            string     `json:"owner"`
	FQDN             string     `json:"fqdn"`
	IPVersion        string     `json:"ip_version"`
	Proxied          bool       `json:"proxied"`
	Status           string     `json:"status"`
	Message          string     `json:"message,omitempty"`
	CurrentIP        string     `json:"current_ip,omitempty"`
	PreviousIPs      []string   `json:"previous_ips"`
	PreviousIPsTotal uint       `json:"previous_ips_total"`
	LastAttempt      *time.Time `json:"last_attempt"`
	LastSuccess      *time.Time `json:"last_success"`
	LastBan          *time.Time `json:"last_ban"`
}

type apiSummary struct {
	Total    uint            `json:"total"`
	Healthy  bool            `json:"healthy"`
	ByStatus map[string]uint `json:"by_status"`
}

type apiRecordsResponse struct {
	Records []apiRecord `json:"records"`
	Summary apiSummary  `json:"summary"`
}

func (h *handlers) apiRecords(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, makeAPIRecordsResponse(h.db.SelectAll()))
}

func (h *handlers) apiRecordsByFQDN(w http.ResponseWriter, r *http.Request) {
	fqdn := chi.URLParam(r, "fqdn")
	var matching []records.Record
	for _, record := range h.db.SelectAll() {
		if strings.EqualFold(record.Provider.BuildDomainName(), fqdn) {
			matching = append(matching, record)
		}
	}
	if len(matching) == 0 {
		http.Error(w, "no record found for "+fqdn, http.StatusNotFound)
		return
	}
	writeJSON(w, makeAPIRecordsResponse(matching))
}

func makeAPIRecordsResponse(recordsList []records.Record) apiRecordsResponse {
	response := apiRecordsResponse{
		Records: make([]apiRecord, len(recordsList)),
		Summary: apiSummary{
			Total:    uint(len(recordsList)),
			Healthy:  true,
			ByStatus: make(map[string]uint),
		},
	}
	for i, record := range recordsList {
		response.Records[i] = toAPIRecord(record)
		response.Summary.ByStatus[string(record.Status)]++
		if record.Status == constants.FAIL || record.Status == constants.UNSET {
			response.Summary.Healthy = false
		}
	}
	return response
}

func toAPIRecord(record records.Record) apiRecord {
	const maxPreviousIPs = 10
	previousIPs := record.History.GetPreviousIPs()
	previousIPStrings := make([]string, min(len(previousIPs), maxPreviousIPs))
	for i := range previousIPStrings {
		previousIPStrings[i] = previousIPs[i].String()
	}

	return apiRecord{
		Domain:           record.Provider.Domain(),
		Owner:            record.Provider.Owner(),
		FQDN:             record.Provider.BuildDomainName(),
		IPVersion:        record.Provider.IPVersion().String(),
		Proxied:          record.Provider.Proxied(),
		Status:           string(record.Status),
		Message:          record.Message,
		CurrentIP:        ipString(record.History.GetCurrentIP()),
		PreviousIPs:      previousIPStrings,
		PreviousIPsTotal: uint(len(previousIPs)),
		LastAttempt:      timePtr(record.Time),
		LastSuccess:      timePtr(record.History.GetSuccessTime()),
		LastBan:          record.LastBan,
	}
}

func ipString(ip netip.Addr) string {
	if !ip.IsValid() {
		return ""
	}
	return ip.String()
}

// timePtr returns nil for the zero time so it is encoded as null
// rather than as a year 1 timestamp.
func timePtr(t time.Time) *time.Time {
	if t.IsZero() {
		return nil
	}
	return &t
}

func writeJSON(w http.ResponseWriter, response apiRecordsResponse) {
	data, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "encoding response: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(data)
}
