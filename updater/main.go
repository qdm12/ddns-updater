package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/kyokomi/emoji"
)

// Global to access with other HTTP GET requests
var rootURL = ""
var fsLocation = ""

type updatesType []updateType
type channelsType struct {
	forceCh chan bool
	quitCh  chan struct{}
}

func init() {
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	fsLocation = filepath.Dir(ex)
}

func healthcheckMode() bool {
	args := os.Args
	if len(args) > 1 {
		if len(args) > 2 {
			log.Fatal(emoji.Sprint(":x:") + " Too many arguments provided")
		}
		if args[1] == "healthcheck" {
			return true
		}
		log.Fatal(emoji.Sprint(":x:") + " Argument 1 can only be 'healthcheck', not " + args[1])
	}
	return false
}

func healthcheck(listeningPort, rootURL string) {
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+listeningPort+rootURL+"healthcheck", nil)
	if err != nil {
		fmt.Println("Can't build HTTP request")
		os.Exit(1)
	}
	client := &http.Client{Timeout: time.Duration(1000) * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		fmt.Println("Can't execute HTTP request")
		os.Exit(1)
	}
	if response.StatusCode != 200 {
		fmt.Println("Status code is " + response.Status)
		os.Exit(1)
	}
	os.Exit(0)
}

func main() {
	if healthcheckMode() {
		listeningPort := getListeningPort()
		rootURL := getRootURL()
		healthcheck(listeningPort, rootURL)
	}
	fmt.Println("#################################")
	fmt.Println("##### DDNS Universal Updater ####")
	fmt.Println("######## by Quentin McGaw #######")
	fmt.Println("######## Give some " + emoji.Sprint(":heart:") + "at #########")
	fmt.Println("# github.com/qdm12/ddns-updater #")
	fmt.Print("#################################\n\n")
	var updates updatesType
	listeningPort, rootURL, delay, updates := getConfig()
	connectivityTest()
	channels := channelsType{
		forceCh: make(chan bool, 1),
		quitCh:  make(chan struct{}),
	}
	go triggerUpdates(&updates, delay, channels.forceCh, channels.quitCh)
	channels.forceCh <- true
	router := httprouter.New()
	router.GET(rootURL, updates.getIndex)
	router.GET(rootURL+"update", channels.getUpdate)
	router.GET(rootURL+"healthcheck", updates.getHealthcheck)
	log.Println("Web UI listening on 0.0.0.0:" + listeningPort + emoji.Sprint(" :ear:"))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+listeningPort, router))
}

func triggerUpdates(updates *updatesType, delay time.Duration, forceCh chan bool, quitCh chan struct{}) {
	ticker := time.NewTicker(delay * time.Second)
	defer func() {
		ticker.Stop()
		close(quitCh)
	}()
	for {
		select {
		case <-ticker.C:
			for i := range *updates {
				go (*updates)[i].update()
			}
		case <-forceCh:
			for i := range *updates {
				go (*updates)[i].update()
			}
		case <-quitCh:
			for {
				allUpdatesFinished := true
				for _, u := range *updates {
					if u.status.code == UPDATING {
						allUpdatesFinished = false
					}
				}
				if allUpdatesFinished {
					break
				}
				log.Println("Waiting for updates to complete...")
				time.Sleep(time.Duration(400) * time.Millisecond)
			}
			ticker.Stop()
			return
		}
	}
}

func (updates *updatesType) getIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// TODO: Forms to change existing updates or add some
	htmlData := updates.toHTML()
	t := template.Must(template.ParseFiles(fsLocation + "/ui/index.html"))
	err := t.ExecuteTemplate(w, "index.html", htmlData) // TODO Without pointer?
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "An error occurred creating this webpage: "+err.Error())
	}
}

func (channels *channelsType) getUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	channels.forceCh <- true
	log.Println("Update started manually " + emoji.Sprint(":repeat:"))
	http.Redirect(w, r, rootURL, 301)
}

func (updates updatesType) getHealthcheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	for _, u := range updates {
		if u.status.code == FAIL {
			log.Println("Responded with error to Healthcheck (" + u.String() + ")")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if u.status.code != UPDATING {
			ips, err := net.LookupIP(u.settings.buildDomainName())
			if err != nil {
				log.Println("Responded with error to Healthcheck (" + err.Error() + ")")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			if len(u.extras.ips) == 0 {
				log.Println("Responded with error to Healthcheck (No set IP address found)")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			for i := range ips {
				if ips[i].String() != u.extras.ips[0] {
					log.Println("Responded with error to Healthcheck (Lookup IP address of " + u.settings.buildDomainName() + " is not equal to " + u.extras.ips[0] + ")")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}
