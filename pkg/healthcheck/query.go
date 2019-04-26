package healthcheck

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

// Mode checks if the program is
// launched to run the Docker internal healthcheck.
func Mode() bool {
	args := os.Args
	if len(args) > 1 && args[1] == "healthcheck" {
		if len(args) > 2 {
			fmt.Println("Too many arguments provided for command healthcheck")
			os.Exit(1)
		}
		return true
	}
	return false
}

// Query sends an HTTP request to
// the other instance of the program's healthcheck
// server route.
func Query() {
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1:9999", nil)
	if err != nil {
		fmt.Print("Cannot build HTTP request")
		os.Exit(1)
	}
	client := &http.Client{Timeout: time.Duration(1000) * time.Millisecond}
	response, err := client.Do(request)
	if err != nil {
		fmt.Print("Cannot execute HTTP request")
		os.Exit(1)
	}
	if response.StatusCode != 200 {
		fmt.Print("Status code is " + response.Status)
		os.Exit(1)
	}
	os.Exit(0)
}
