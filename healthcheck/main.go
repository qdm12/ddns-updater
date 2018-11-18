package main

import (
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const defaultListeningPort = "8000"
const defaultRootURL = "/"

// Error codes:
// 1: Error in main server
// 2: Error in communicating with main server
// 3: Error in creating HTTP request
// 4: Error in parsing parameters

func main() {
	listeningPort := os.Getenv("LISTENINGPORT")
	if len(listeningPort) == 0 {
		listeningPort = defaultListeningPort
	} else {
		value, err := strconv.ParseInt(listeningPort, 10, 64)
		if err != nil || value < 1 || value > 65535 {
			os.Exit(4)
		}
	}
	rootURL := os.Getenv("ROOTURL")
	if len(rootURL) == 0 {
		rootURL = defaultRootURL
	} else if strings.ContainsAny(rootURL, " .?~#") {
		os.Exit(4)
	}
	if rootURL[len(rootURL)-1] != '/' {
		rootURL += "/"
	}
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:"+listeningPort+rootURL+"healthcheck", nil)
	if err != nil {
		os.Exit(3)
	}
	client := &http.Client{Timeout: time.Duration(7000) * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		os.Exit(2)
	}
	if response.StatusCode != 200 {
		os.Exit(1)
	}
}
