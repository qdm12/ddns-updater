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

type updatesType []updateType
type envType struct {
	fsLocation string
	rootURL    string
	delay      time.Duration
	updates    updatesType
	forceCh    chan struct{}
	quitCh     chan struct{}
	db         *DB
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
	listeningPort := getListeningPort()
	if healthcheckMode() {
		rootURL := getRootURL()
		healthcheck(listeningPort, rootURL)
	}
	fmt.Println("#################################")
	fmt.Println("##### DDNS Universal Updater ####")
	fmt.Println("######## by Quentin McGaw #######")
	fmt.Println("######## Give some " + emoji.Sprint(":heart:") + "at #########")
	fmt.Println("# github.com/qdm12/ddns-updater #")
	fmt.Print("#################################\n\n")
	var env envType
	env.rootURL = getRootURL()
	env.delay = getDelay()
	env.updates = getUpdates()
	env.forceCh = make(chan struct{})
	env.quitCh = make(chan struct{})
	ex, err := os.Executable()
	if err != nil {
		log.Fatal(err)
	}
	env.fsLocation = filepath.Dir(ex)
	dataDir := getDataDir(env.fsLocation)
	env.db, err = initializeDatabase(dataDir)
	if err != nil {
		log.Fatal(err)
	}
	for i := range env.updates {
		u := &env.updates[i]
		var err error
		u.m.Lock()
		u.extras.ips, u.extras.tSuccess, err = env.db.getIps(u.settings.domain, u.settings.host)
		log.Println(u.extras.ips)
		u.m.Unlock()
		if err != nil {
			log.Fatal(err)
		}
	}
	connectivityTest()
	go triggerUpdates(&env)
	env.forceCh <- struct{}{}
	router := httprouter.New()
	router.GET(env.rootURL, env.getIndex)
	router.GET(env.rootURL+"update", env.getUpdate)
	router.GET(env.rootURL+"healthcheck", env.getHealthcheck)
	log.Println("Web UI listening on 0.0.0.0:" + listeningPort + emoji.Sprint(" :ear:"))
	log.Fatal(http.ListenAndServe("0.0.0.0:"+listeningPort, router))
}

func triggerUpdates(env *envType) {
	ticker := time.NewTicker(env.delay * time.Second)
	defer func() {
		ticker.Stop()
		close(env.quitCh)
	}()
	for {
		select {
		case <-ticker.C:
			for i := range env.updates {
				go env.update(i)
			}
		case <-env.forceCh:
			for i := range env.updates {
				go env.update(i)
			}
		case <-env.quitCh:
			for {
				allUpdatesFinished := true
				for i := range env.updates {
					u := &env.updates[i]
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

func (env *envType) getIndex(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	// TODO: Forms to change existing updates or add some
	htmlData := env.updates.toHTML()
	t := template.Must(template.ParseFiles(env.fsLocation + "/ui/index.html"))
	err := t.ExecuteTemplate(w, "index.html", htmlData) // TODO Without pointer?
	if err != nil {
		log.Println(err)
		fmt.Fprint(w, "An error occurred creating this webpage: "+err.Error())
	}
}

func (env *envType) getUpdate(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	env.forceCh <- struct{}{}
	log.Println("Update started manually " + emoji.Sprint(":repeat:"))
	http.Redirect(w, r, env.rootURL, 301)
}

func (env *envType) getHealthcheck(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	for i := range env.updates {
		u := &env.updates[i]
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
