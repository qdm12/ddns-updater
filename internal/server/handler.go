package server

import (
	"fmt"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/logging"
)

func newHandler(rootURL, uiDir string, db data.Database,
	logger logging.Logger, forceUpdate chan<- struct{}) http.Handler {
	return &handler{
		rootURL: rootURL,
		uiDir:   uiDir,
		db:      db,
		logger:  logger, // TODO log middleware
		// TODO build information
		timeNow:     time.Now,
		forceUpdate: forceUpdate,
	}
}

type handler struct {
	// Configuration
	rootURL string
	uiDir   string
	// Channels
	forceUpdate chan<- struct{}
	// Objects and mock functions
	db      data.Database
	logger  logging.Logger
	timeNow func() time.Time
}

func (h *handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.logger.Info("HTTP %s %s", r.Method, r.RequestURI)

	r.RequestURI = strings.TrimPrefix(r.RequestURI, h.rootURL)
	switch {
	case r.Method == http.MethodGet && r.RequestURI == h.rootURL+"/":
		t := template.Must(template.ParseFiles(h.uiDir + "/index.html"))
		var htmlData models.HTMLData
		for _, record := range h.db.SelectAll() {
			row := record.HTML(h.timeNow())
			htmlData.Rows = append(htmlData.Rows, row)
		}
		if err := t.ExecuteTemplate(w, "index.html", htmlData); err != nil {
			h.logger.Warn(err)
			fmt.Fprint(w, "An error occurred creating this webpage")
		}
	case r.Method == http.MethodGet && r.RequestURI == h.rootURL+"/update":
		h.logger.Info("Update started manually")
		h.forceUpdate <- struct{}{}
		http.Redirect(w, r, h.rootURL, 301)
	default:
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
	}
}
