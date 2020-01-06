package router

import (
	"fmt"
	"net/http"
	"text/template"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/qdm12/golibs/admin"
	"github.com/qdm12/golibs/logging"

	"github.com/julienschmidt/httprouter"
)

type indexParamsType struct {
	dir            string
	recordsConfigs []models.RecordConfigType
}

type updateParamsType struct {
	rootURL string
	forceCh chan struct{}
}

// CreateRouter returns a router with all the necessary routes configured
func CreateRouter(rootURL, dir string, forceCh chan struct{}, recordsConfigs []models.RecordConfigType, gotify admin.Gotify) *httprouter.Router {
	indexParams := indexParamsType{
		dir:            dir,
		recordsConfigs: recordsConfigs,
	}
	updateParams := updateParamsType{
		rootURL: rootURL,
		forceCh: forceCh,
	}
	router := httprouter.New()
	router.GET(rootURL+"/", indexParams.get)
	router.GET(rootURL+"/update", updateParams.get)
	return router
}

func (params *indexParamsType) get(w http.ResponseWriter, _ *http.Request, _ httprouter.Params) {
	// TODO: Forms to change existing updates or add some
	t := template.Must(template.ParseFiles(params.dir + "/ui/index.html"))
	htmlData := models.ToHTML(params.recordsConfigs)
	err := t.ExecuteTemplate(w, "index.html", htmlData) // TODO Without pointer?
	if err != nil {
		logging.Warn(err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "An error occurred creating this webpage")
	}
}

func (params *updateParamsType) get(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	params.forceCh <- struct{}{}
	logging.Info("Update started manually")
	http.Redirect(w, r, params.rootURL, 301)
}
