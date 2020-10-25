package handlers

import (
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/logging"
)

// MakeHandler returns a router with all the necessary routes configured.
func MakeHandler(rootURL, uiDir string, db data.Database, logger logging.Logger,
	forceUpdate func(), timeNow func() time.Time) http.HandlerFunc {
	logger = logger.WithPrefix("http server: ")
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Info("HTTP %s %s", r.Method, r.RequestURI)
		switch {
		case r.Method == http.MethodGet && r.RequestURI == rootURL+"/":
			t := template.Must(template.ParseFiles(uiDir + "/index.html"))
			var htmlData models.HTMLData
			for _, record := range db.SelectAll() {
				row := record.HTML(timeNow())
				htmlData.Rows = append(htmlData.Rows, row)
			}
			if err := t.ExecuteTemplate(w, "index.html", htmlData); err != nil {
				logger.Warn(err)
				fmt.Fprint(w, "An error occurred creating this webpage")
			}
		case r.Method == http.MethodGet && r.RequestURI == rootURL+"/update":
			logger.Info("Update started manually")
			forceUpdate()
			http.Redirect(w, r, rootURL, 301)
		}
	}
}
