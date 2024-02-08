package models

type BuildInformation struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"buildDate"`
}

func (b BuildInformation) VersionString() string {
	if b.Version != "latest" {
		return b.Version
	}
	const commitShortHashLength = 7
	if len(b.Commit) != commitShortHashLength {
		return "latest"
	}
	return b.Version + "-" + b.Commit[:7]
}
