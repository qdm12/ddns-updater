package update

import (
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	domain = os.Getenv("ddns_domain")
	host   = os.Getenv("ddns_host")
	token  = os.Getenv("ddns_token")
)

func Test_updateDnsPod(t *testing.T) {
	client := http.DefaultClient
	ip, err := updateDnsPod(
		client,
		domain,
		host,
		token,
		"120.121.121.123",
	)
	if assert.NoError(t, err) {
		t.Log(ip)
	}
}
