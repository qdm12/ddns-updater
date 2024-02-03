package models

type BuildInformation struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"buildDate"`
}
