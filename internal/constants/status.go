package constants

import "github.com/qdm12/ddns-updater/internal/models"

const (
	FAIL     models.Status = "failure"
	SUCCESS  models.Status = "success"
	UPTODATE models.Status = "up to date"
	UPDATING models.Status = "updating"
)

const (
	HTML_FAIL     string = `<font color="red">Failure</font>`
	HTML_SUCCESS  string = `<font color="green">Success</font>`
	HTML_UPTODATE string = `<font color="#00CC66">Up to date</font>`
	HTML_UPDATING string = `<font color="yellow">Updating</font>`
)
