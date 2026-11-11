package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	FAIL     models.Status = "failure"
	SUCCESS  models.Status = "success"
	UPTODATE models.Status = "up to date"
	UPDATING models.Status = "updating"
	UNSET    models.Status = "unset"
	// PUBLICIPUNKNOWN is the status of a record for which no update could be
	// attempted because its public IP address could not be found. Contrary to
	// FAIL it is not the result of an update: it is re-evaluated at every
	// update cycle until the public IP address is found.
	PUBLICIPUNKNOWN models.Status = "public IP unknown"
)
