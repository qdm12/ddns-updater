package errors

import "errors"

var (
	ErrEmptyApiKey             = errors.New("empty API key")
	ErrEmptyAppKey             = errors.New("empty app key")
	ErrEmptyConsumerKey        = errors.New("empty consumer key")
	ErrEmptyEmail              = errors.New("empty email")
	ErrEmptyKey                = errors.New("empty key")
	ErrEmptyName               = errors.New("empty name")
	ErrEmptyPassword           = errors.New("empty password")
	ErrEmptyApiSecret          = errors.New("empty api secret")
	ErrEmptySecret             = errors.New("empty secret")
	ErrEmptyToken              = errors.New("empty token")
	ErrEmptyTTL                = errors.New("TTL is not set")
	ErrEmptyUsername           = errors.New("empty username")
	ErrEmptyZoneIdentifier     = errors.New("empty zone identifier")
	ErrHostOnlyAt              = errors.New(`host can only be "@"`)
	ErrHostOnlySubdomain       = errors.New("host can only be a subdomain")
	ErrHostWildcard            = errors.New(`host cannot be a "*"`)
	ErrIPv6NotSupported        = errors.New("IPv6 is not supported by this provider")
	ErrMalformedEmail          = errors.New("malformed email address")
	ErrMalformedKey            = errors.New("malformed key")
	ErrMalformedPassword       = errors.New("malformed password")
	ErrMalformedToken          = errors.New("malformed token")
	ErrMalformedUsername       = errors.New("malformed username")
	ErrMalformedUserServiceKey = errors.New("malformed user service key")
)
