package server

import (
	"fmt"
	"net/http"
	"text/template"

	"ddns-updater/pkg/models"
	"ddns-updater/pkg/network"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

type healthcheckParamsType struct {
	recordsConfigs []models.RecordConfigType
}

type indexParamsType struct {
	dir            string
	recordsConfigs []models.RecordConfigType
}

type updateParamsType struct {
	rootURL string
	forceCh chan struct{}
}

// CreateRouter returns a router with all the necessary routes configured
func CreateRouter(rootURL, dir string, forceCh chan struct{}, recordsConfigs []models.RecordConfigType) *httprouter.Router {
	healthcheckParams := healthcheckParamsType{
		recordsConfigs: recordsConfigs,
	}
	indexParams := indexParamsType{
		dir:            dir,
		recordsConfigs: recordsConfigs,
	}
	updateParams := updateParamsType{
		rootURL: rootURL,
		forceCh: forceCh,
	}
	router := httprouter.New()
	router.GET(rootURL+"/healthcheck", healthcheckParams.get)
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
		zap.S().Warn(err)
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "An error occurred creating this webpage")
	}
}

func (params *healthcheckParamsType) get(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	clientIP, err := network.GetClientIP(r)
	if err != nil {
		zap.S().Infof("Cannot detect client IP: %s", err)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	if clientIP != "127.0.0.1" && clientIP != "::1" {
		zap.S().Infof("IP address %s tried to perform the healthcheck", clientIP)
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	err = healthcheckHandler(params.recordsConfigs)
	if err != nil {
		zap.S().Warnf("Responded with error to healthcheck: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (params *updateParamsType) get(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	params.forceCh <- struct{}{}
	zap.S().Info("Update started manually")
	http.Redirect(w, r, params.rootURL, 301)
}
