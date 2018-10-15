package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"text/template"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/kyokomi/emoji"
)

const httpGetTimeout = 10000 // 10 seconds

// Global to access with other HTTP GET requests
var rootURL = ""

type Updates []*updateType
type Channels struct {
	forceCh chan bool
	quitCh  chan struct{}
}

func main() {
	fmt.Println("#################################")
	fmt.Println("##### DDNS Universal Updater ####")
	fmt.Println("######## by Quentin McGaw #######")
	fmt.Println("######## Give some " + emoji.Sprint(":heart:") + "at ########")
	fmt.Println("# github.com/qdm12/ddns-updater #")
	fmt.Print("#################################\n\n")
	var updates Updates
	listeningPort, rootURL, delay, updates := parseEnvConfig()
	connectivityTest()
	channels := Channels{
		forceCh: make(chan bool, 1),
		quitCh:  make(chan struct{}),
	}
	ticker := time.NewTicker(delay * time.Second)
	defer func() {
		ticker.Stop()
		close(channels.quitCh)
	}()
	go func() {
		for {
			select {
			case <-ticker.C:
				for i := range updates {
					go update(updates[i])
				}
			case <-channels.forceCh:
				for i := range updates {
					go update(updates[i])
				}
			case <-channels.quitCh:
				for {
					allUpdatesFinished := true
					for i := range updates {
						if updates[i].status.code == UPDATING {
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
	}()
	channels.forceCh <- true
	router := httprouter.New()
	router.GET(rootURL, updates.getIndex)
	router.GET(rootURL+"update", channels.getUpdate)
	router.GET(rootURL+"healthcheck", updates.getHealthcheck)
	log.Println("Web UI listening on 0.0.0.0:" + listeningPort + emoji.Sprint(" :ear:"))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+listeningPort, router))
}

func (updates *Updates) getIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// TODO: Forms to change existing updates or add some
	htmlData := updatesToHtml(updates)
	t := template.Must(template.ParseFiles("/index.html"))
	err := t.ExecuteTemplate(w, "index.html", htmlData) // TODO Without pointer?
	if err != nil {
		log.Println(err.Error())
		fmt.Fprint(w, "An error occurred creating this webpage: "+err.Error())
	}
}

func (channels *Channels) getUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	channels.forceCh <- true
	log.Println("Update started manually" + emoji.Sprint(" :repeat:"))
	http.Redirect(w, r, rootURL, 301)
}

func (updates *Updates) getHealthcheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	for _, u := range *updates {
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
