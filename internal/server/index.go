package server

import (
	"net/http"

	"github.com/qdm12/ddns-updater/internal/constants"
	"github.com/qdm12/ddns-updater/internal/models"
)

func (h *handlers) index(w http.ResponseWriter, r *http.Request) {
	// Prevent caching to ensure status updates are always fresh
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")

	var htmlData models.HTMLData
	successCount := 0
	errorCount := 0
	updatingCount := 0

	records := h.db.SelectAll()
	for _, record := range records {
		row := record.HTML(h.timeNow())
		htmlData.Rows = append(htmlData.Rows, row)

		// Count statuses for summary statistics
		switch record.Status {
		case constants.SUCCESS, constants.UPTODATE:
			successCount++
		case constants.FAIL:
			errorCount++
		case constants.UPDATING:
			updatingCount++
		}
	}

	// Set summary statistics
	htmlData.TotalDomains = len(records)
	htmlData.SuccessCount = successCount
	htmlData.ErrorCount = errorCount
	htmlData.UpdatingCount = updatingCount
	htmlData.LastUpdate = h.timeNow().Format("15:04:05")

	// Fetch public IP addresses
	if h.ipGetter != nil {
		ipv4, err := h.ipGetter.IP4(r.Context())
		if err == nil && ipv4.IsValid() {
			htmlData.PublicIPv4 = ipv4.String()
		}

		ipv6, err := h.ipGetter.IP6(r.Context())
		if err == nil && ipv6.IsValid() {
			htmlData.PublicIPv6 = ipv6.String()
		}
	}

	err := h.indexTemplate.ExecuteTemplate(w, "index.html", htmlData)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed generating webpage: "+err.Error())
	}
}
