package config

import (
	"fmt"
	"net/url"
	"os"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/reader"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Health struct {
	ServerAddress         *string
	HealthchecksioBaseURL string
	HealthchecksioUUID    *string
}

func (h *Health) SetDefaults() {
	h.ServerAddress = gosettings.DefaultPointer(h.ServerAddress, "127.0.0.1:9999")
	h.HealthchecksioBaseURL = gosettings.DefaultComparable(h.HealthchecksioBaseURL, "https://hc-ping.com")
	h.HealthchecksioUUID = gosettings.DefaultPointer(h.HealthchecksioUUID, "")
}

func (h Health) Validate() (err error) {
	err = validate.ListeningAddress(*h.ServerAddress, os.Getuid())
	if err != nil {
		return fmt.Errorf("server listening address: %w", err)
	}

	_, err = url.Parse(h.HealthchecksioBaseURL)
	if err != nil {
		return fmt.Errorf("healthchecks.io base URL: %w", err)
	}

	return nil
}

func (h Health) String() string {
	return h.toLinesNode().String()
}

func (h Health) toLinesNode() *gotree.Node {
	node := gotree.New("Health")
	node.Appendf("Server listening address: %s", *h.ServerAddress)
	if *h.HealthchecksioUUID != "" {
		node.Appendf("Healthchecks.io base URL: %s", h.HealthchecksioBaseURL)
		node.Appendf("Healthchecks.io UUID: %s", *h.HealthchecksioUUID)
	}
	return node
}

func (h *Health) Read(reader *reader.Reader) {
	h.ServerAddress = reader.Get("HEALTH_SERVER_ADDRESS")
	h.HealthchecksioBaseURL = reader.String("HEALTH_HEALTHCHECKSIO_BASE_URL")
	h.HealthchecksioUUID = reader.Get("HEALTH_HEALTHCHECKSIO_UUID")
}
