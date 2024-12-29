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

	currentVersion := getCurrentVersion()
	latestVersion, err := getLatestRelease()
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed getting latest release: "+err.Error())
		return
	}
	htmlData.Version = currentVersion
	if currentVersion != latestVersion {
		htmlData.UpdateAvailable = true
		htmlData.LatestVersion = latestVersion
	}

	err = h.indexTemplate.ExecuteTemplate(w, "index.html", htmlData)
	if err != nil {
		httpError(w, http.StatusInternalServerError, "failed generating webpage: "+err.Error())
	}
}
