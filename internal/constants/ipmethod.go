package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	IPMETHODPROVIDER models.IPMethod = "provider"
	IPMETHODGOOGLE   models.IPMethod = "google"
	IPMETHODOPENDNS  models.IPMethod = "opendns"
)

func IPMethodChoices() (choices []models.IPMethod) {
	return []models.IPMethod{
		IPMETHODPROVIDER,
		IPMETHODGOOGLE,
		IPMETHODOPENDNS,
	}
}
