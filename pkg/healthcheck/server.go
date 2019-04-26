package healthcheck

import (
	"net"
	"net/http"
	"fmt"

	"github.com/julienschmidt/httprouter"

	"ddns-updater/pkg/logging"
	"ddns-updater/pkg/models"
)

type paramsType struct {
	recordsConfigs []models.RecordConfigType
}

// Serve healthcheck HTTP requests and listens on
// localhost:9999 only.
func Serve(recordsConfigs []models.RecordConfigType) {
	params := paramsType{
		recordsConfigs: recordsConfigs,
	}
	localRouter := httprouter.New()
	localRouter.GET("/", params.get)
	logging.Info("Private server listening on 127.0.0.1:9999")
	err := http.ListenAndServe("127.0.0.1:9999", localRouter)
	if err != nil {
		logging.Fatal("%s", err)
	}
}

func (params *paramsType) get(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	err := getHandler(params.recordsConfigs)
	if err != nil {
		logging.Warn("Responded with error to healthcheck: %s", err)
		w.WriteHeader(http.StatusInternalServerError)
	} else {
		w.WriteHeader(http.StatusOK)
	}
}

func getHandler(recordsConfigs []models.RecordConfigType) error {
	for i := range recordsConfigs {
		if recordsConfigs[i].Status.Code == models.FAIL {
			return fmt.Errorf("%s", recordsConfigs[i].String())
		}
		if recordsConfigs[i].Status.Code != models.UPDATING {
			ips, err := net.LookupIP(recordsConfigs[i].Settings.BuildDomainName())
			if err != nil {
				return fmt.Errorf("%s", err)
			}
			if len(recordsConfigs[i].History.IPs) == 0 {
				return fmt.Errorf("no set IP address found")
			}
			for i := range ips {
				if ips[i].String() != recordsConfigs[i].History.IPs[0] {
					return fmt.Errorf(
						"lookup IP address of %s is not %s",
						recordsConfigs[i].Settings.BuildDomainName(),
						recordsConfigs[i].History.IPs[0],
					)
				}
			}
		}
	}
	return nil
}
