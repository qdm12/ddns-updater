package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	FAIL     models.Status = "failure"
	SUCCESS  models.Status = "success"
	UPTODATE models.Status = "up to date"
	UPDATING models.Status = "updating"
)

const (
	HTML_FAIL     string = `<font color="red"><b>Failure</b></font>`
	HTML_SUCCESS  string = `<font color="green"><b>Success</b></font>`
	HTML_UPTODATE string = `<font color="#00CC66"><b>Up to date</b></font>`
	HTML_UPDATING string = `<font color="orange"><b>Updating</b></font>`
)
