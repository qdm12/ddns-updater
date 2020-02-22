package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	PROVIDER models.IPMethod = "provider"
	GOOGLE   models.IPMethod = "google"
	OPENDNS  models.IPMethod = "opendns"
)

func IPMethodChoices() (choices []models.IPMethod) {
	return []models.IPMethod{
		PROVIDER,
		GOOGLE,
		OPENDNS,
	}
}
