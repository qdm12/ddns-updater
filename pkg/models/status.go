package models

import "time"

type statusCode uint8

// Update possible status codes: FAIL, SUCCESS, UPTODATE or UPDATING
const (
	FAIL statusCode = iota
	SUCCESS
	UPTODATE
	UPDATING
)

func (code *statusCode) string() (s string) {
	switch *code {
	case SUCCESS:
		return "Success"
	case FAIL:
		return "Failure"
	case UPTODATE:
		return "Up to date"
	case UPDATING:
		return "Already updating..."
	default:
		return "Unknown status code!"
	}
}

func (code *statusCode) toHTML() (s string) {
	switch *code {
	case SUCCESS:
		return `<font color="green">Success</font>`
	case FAIL:
		return `<font color="red">Failure</font>`
	case UPTODATE:
		return `<font color="#00CC66">Up to date</font>`
	case UPDATING:
		return `<font color="orange">Already updating...</font>`
	default:
		return `<font color="red">Unknown status code!</font>`
	}
}

type statusType struct {
	Code    statusCode
	Message string
	Time    time.Time
}

func (status *statusType) string() (s string) {
	s += status.Code.string()
	if status.Message != "" {
		s += " (" + status.Message + ")"
	}
	s += " at " + status.Time.Format("2006-01-02 15:04:05 MST")
	return s
}

func (status *statusType) toHTML() (s string) {
	s += status.Code.toHTML()
	if status.Message != "" {
		s += " (" + status.Message + ")"
	}
	s += ", " + time.Since(status.Time).Round(time.Second).String() + " ago"
	return s
}
