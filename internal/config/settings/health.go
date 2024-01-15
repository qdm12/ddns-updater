package settings

import (
	"fmt"
	"os"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/validate"
	"github.com/qdm12/gotree"
)

type Health struct {
	ServerAddress      *string
	HealthchecksioUUID *string
}

func (h *Health) SetDefaults() {
	h.ServerAddress = gosettings.DefaultPointer(h.ServerAddress, "127.0.0.1:9999")
	h.HealthchecksioUUID = gosettings.DefaultPointer(h.HealthchecksioUUID, "")
}

func (h Health) mergeWith(other Health) (merged Health) {
	merged.ServerAddress = gosettings.MergeWithPointer(h.ServerAddress, other.ServerAddress)
	merged.HealthchecksioUUID = gosettings.MergeWithPointer(h.HealthchecksioUUID, other.HealthchecksioUUID)
	return merged
}

func (h Health) Validate() (err error) {
	err = validate.ListeningAddress(*h.ServerAddress, os.Getuid())
	if err != nil {
		return fmt.Errorf("server listening address: %w", err)
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
		node.Appendf("Healthchecks.io UUID: %s", *h.HealthchecksioUUID)
	}
	return node
}
