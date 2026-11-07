package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	FAIL     models.Status = "failure"
	SUCCESS  models.Status = "success"
	UPTODATE models.Status = "up to date"
	UPDATING models.Status = "updating"
	UNSET    models.Status = "unset"
	// NOIP is the status of a record for which no update was attempted
	// yet, because its public IP address could not be found.
	NOIP models.Status = "no IP"
)
