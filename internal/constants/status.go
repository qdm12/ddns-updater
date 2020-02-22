package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	FAIL     models.Status = "failure"
	SUCCESS  models.Status = "success"
	UPTODATE models.Status = "up to date"
	UPDATING models.Status = "updating"
)
