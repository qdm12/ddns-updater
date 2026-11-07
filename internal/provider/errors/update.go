package errors

import "errors"

var (
	ErrBannedAbuse               = errors.New("banned due to abuse")
	ErrConflictingRecord         = errors.New("conflicting record")
	ErrReceivedNoResult          = errors.New("received no result in response")
	ErrRecordNotFound            = errors.New("record not found")
	ErrRecordResourceSetNotFound = errors.New("record resource set not found")
)
