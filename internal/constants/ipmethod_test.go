package constants

import (
	"testing"

	"github.com/qdm12/ddns-updater/internal/models"
	"github.com/stretchr/testify/assert"
)

func Test_IPMethodChoices(t *testing.T) {
	t.Parallel()
	choices := IPMethodChoices()
	assert.ElementsMatch(t, []models.IPMethod{"ipinfo", "ipify", "ipify6", "provider", "cycle", "opendns", "ifconfig", "ddnss", "ddnss4", "ddnss6"}, choices)
}

func Test_IPMethodExternalChoices(t *testing.T) {
	t.Parallel()
	choices := IPMethodExternalChoices()
	assert.ElementsMatch(t, []models.IPMethod{"ipinfo", "ipify", "ipify6", "ifconfig", "opendns", "ddnss", "ddnss4", "ddnss6"}, choices)
}
