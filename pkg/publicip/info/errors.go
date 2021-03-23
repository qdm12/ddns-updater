package info

import "errors"

var (
	ErrTooManyRequests = errors.New("too many requests sent")
	ErrBadHTTPStatus   = errors.New("bad HTTP status received")
)
