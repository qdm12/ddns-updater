package settings

import (
	"fmt"
	"os"

	"github.com/qdm12/gosettings"
	"github.com/qdm12/gosettings/validate"
)

type Health struct {
	ServerAddress *string
}

func (h *Health) SetDefaults() {
	h.ServerAddress = gosettings.DefaultPointer(h.ServerAddress, "127.0.0.1:9999")
}

func (h Health) mergeWith(other Health) (merged Health) {
	merged.ServerAddress = gosettings.MergeWithPointer(h.ServerAddress, other.ServerAddress)
	return merged
}

func (h Health) Validate() (err error) {
	err = validate.ListeningAddress(*h.ServerAddress, os.Getuid())
	if err != nil {
		return fmt.Errorf("server listening address: %w", err)
	}

	return nil
}
