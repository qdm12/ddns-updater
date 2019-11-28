package models

import (
	"fmt"
	"sync"
	"time"
)

type statusCode uint8

// Update possible status codes: FAIL, SUCCESS, UPTODATE or UPDATING
const (
	FAIL statusCode = iota
	SUCCESS
	UPTODATE
)

func (code *statusCode) String() (s string) {
	switch *code {
	case SUCCESS:
		return "Success"
	case FAIL:
		return "Failure"
	case UPTODATE:
		return "Up to date"
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
	default:
		return `<font color="red">Unknown status code!</font>`
	}
}

type statusType struct {
	code    statusCode
	message string
	time    time.Time
	sync.RWMutex
}

func (status *statusType) SetTime(t time.Time) {
	status.Lock()
	defer status.Unlock()
	status.time = t
}

func (status *statusType) SetCode(code statusCode) {
	status.Lock()
	defer status.Unlock()
	status.code = code
}

func (status *statusType) SetMessage(format string, a ...interface{}) {
	status.Lock()
	defer status.Unlock()
	status.message = fmt.Sprintf(format, a...)
}

func (status *statusType) GetTime() time.Time {
	status.RLock()
	defer status.RUnlock()
	return status.time
}

func (status *statusType) GetCode() statusCode {
	status.RLock()
	defer status.RUnlock()
	return status.code
}

func (status *statusType) GetMessage() string {
	status.RLock()
	defer status.RUnlock()
	return status.message
}

func (status *statusType) String() (s string) {
	status.RLock()
	defer status.RUnlock()
	s += status.code.String()
	if status.message != "" {
		s += " (" + status.message + ")"
	}
	s += " at " + status.time.Format("2006-01-02 15:04:05 MST")
	return s
}

func (status *statusType) toHTML() (s string) {
	status.RLock()
	defer status.RUnlock()
	s += status.code.toHTML()
	if status.message != "" {
		s += " (" + status.message + ")"
	}
	s += ", " + time.Since(status.time).Round(time.Second).String() + " ago"
	return s
}
