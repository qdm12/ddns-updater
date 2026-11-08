package server

import (
	"html/template"
	"net/http"

	"github.com/qdm12/ddns-updater/internal/models"
)

// indexRow is an HTMLRow with its fields typed as template.HTML, because they
// contain markup built by the provider and record code: links and status
// spans. The templates are parsed with html/template, which escapes a plain
// string, and the table would otherwise show the markup instead of rendering
// it.
type indexRow struct {
	Domain      template.HTML
	Owner       template.HTML
	Provider    template.HTML
	IPVersion   template.HTML
	Status      template.HTML
	CurrentIP   template.HTML
	PreviousIPs template.HTML
}

// indexData is the data passed to the index template.
type indexData struct {
	Rows []indexRow
}

// newIndexRow marks the fields as HTML. The values are produced by the
// provider and record code, not by a user submitting a form, so they are
// trusted in the same way they were with text/template.
//
//nolint:gosec // the fields are markup built by the provider and record code
func newIndexRow(row models.HTMLRow) indexRow {
	return indexRow{
		Domain:      template.HTML(row.Domain),
		Owner:       template.HTML(row.Owner),
		Provider:    template.HTML(row.Provider),
		IPVersion:   template.HTML(row.IPVersion),
		Status:      template.HTML(row.Status),
		CurrentIP:   template.HTML(row.CurrentIP),
		PreviousIPs: template.HTML(row.PreviousIPs),
	}
}

func (h *handlers) index(w http.ResponseWriter, _ *http.Request) {
	var data indexData
	for _, record := range h.db.SelectAll() {
		row := record.HTML(h.timeNow())
		data.Rows = append(data.Rows, newIndexRow(row))
	}
	err := h.templates.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed generating webpage: "+err.Error())
	}
}
