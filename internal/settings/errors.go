package settings

import "errors"

var (
	ErrAbuse = errors.New("banned due to abuse")
	ErrAuth  = errors.New("bad authentication")
)
