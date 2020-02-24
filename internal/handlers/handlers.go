package handlers

import (
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/qdm12/ddns-updater/internal/data"
	"github.com/qdm12/ddns-updater/internal/html"
	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/logging"
)

// Handler contains a handler function
type Handler interface {
	GetHandlerFunc() http.HandlerFunc
}

type handler struct {
	rootURL     string
	UIDir       string
	db          data.Database
	logger      logging.Logger
	forceUpdate func()
	onError     func(err error)
	getTime     func() time.Time
}

// NewHandler returns a Handler object
func NewHandler(rootURL, UIDir string, db data.Database, logger logging.Logger,
	forceUpdate func(), onError func(err error)) Handler {
	return &handler{
		rootURL:     rootURL,
		UIDir:       UIDir,
		db:          db,
		logger:      logger,
		forceUpdate: forceUpdate,
		onError:     onError,
		getTime:     time.Now,
	}
}

// GetHandlerFunc returns a router with all the necessary routes configured
func (h *handler) GetHandlerFunc() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.logger.Info("received HTTP request at %s", r.RequestURI)
		switch {
		case r.Method == http.MethodGet && r.RequestURI == h.rootURL+"/":
			// TODO: Forms to change existing updates or add some
			t := template.Must(template.ParseFiles(h.UIDir + "/ui/index.html"))
			var htmlData models.HTMLData
			for _, record := range h.db.SelectAll() {
				row := html.ConvertRecord(record, h.getTime())
				htmlData.Rows = append(htmlData.Rows, row)
			}
			if err := t.ExecuteTemplate(w, "index.html", htmlData); err != nil {
				h.logger.Warn(err)
				fmt.Fprint(w, "An error occurred creating this webpage")
			}
		case r.Method == http.MethodGet && r.RequestURI == h.rootURL+"/update":
			h.logger.Info("Update started manually")
			h.forceUpdate()
			http.Redirect(w, r, h.rootURL, 301)
		}
	}
}
