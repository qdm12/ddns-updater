package server

import (
	"net/http"

	"github.com/qdm12/ddns-updater/internal/models"
)

func (h *handlers) index(w http.ResponseWriter, _ *http.Request) {
	var htmlData models.HTMLData
	for _, record := range h.db.SelectAll() {
		row := record.HTML(h.timeNow())
		htmlData.Rows = append(htmlData.Rows, row)
	}
	data := struct {
		models.HTMLData
		PublicIPCheck bool
		ConfigEditor  bool
	}{
		HTMLData:      htmlData,
		PublicIPCheck: h.publicIPChecker != nil,
		ConfigEditor:  h.configStore != nil,
	}
	err := h.indexTemplate.ExecuteTemplate(w, "index.html", data)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed generating webpage: "+err.Error())
	}
}
