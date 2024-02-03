package errors

import "errors"

var (
	ErrAccessKeyIDNotSet      = errors.New("access key id is not set")
	ErrAccessKeySecretNotSet  = errors.New("key secret is not set")
	ErrAPIKeyNotSet           = errors.New("API key is not set")
	ErrAPISecretNotSet        = errors.New("API secret is not set")
	ErrAppKeyNotSet           = errors.New("app key is not set")
	ErrConsumerKeyNotSet      = errors.New("consumer key is not set")
	ErrCredentialsNotSet      = errors.New("credentials are not set")
	ErrCustomerNumberNotSet   = errors.New("customer number is not set")
	ErrEmailNotSet            = errors.New("email is not set")
	ErrEmailNotValid          = errors.New("email address is not valid")
	ErrGCPProjectNotSet       = errors.New("GCP project is not set")
	ErrHostOnlySubdomain      = errors.New("host can only be a subdomain")
	ErrHostWildcard           = errors.New(`host cannot be a "*"`)
	ErrIPv4KeyNotSet          = errors.New("IPv4 key is not set")
	ErrIPv6KeyNotSet          = errors.New("IPv6 key is not set")
	ErrKeyNotSet              = errors.New("key is not set")
	ErrKeyNotValid            = errors.New("key is not valid")
	ErrNameNotSet             = errors.New("name is not set")
	ErrPasswordNotSet         = errors.New("password is not set")
	ErrPasswordNotValid       = errors.New("password is not valid")
	ErrParametersNotValid     = errors.New("username, password or host incorrect")
	ErrSecretNotSet           = errors.New("secret is not set")
	ErrSuccessRegexNotSet     = errors.New("success regex is not set")
	ErrTokenNotSet            = errors.New("token is not set")
	ErrTokenNotValid          = errors.New("token is not valid")
	ErrTTLNotSet              = errors.New("TTL is not set")
	ErrTTLTooLow              = errors.New("TTL is too low")
	ErrURLNotHTTPS            = errors.New("url is not https")
	ErrURLNotSet              = errors.New("url is not set")
	ErrUsernameNotSet         = errors.New("username is not set")
	ErrUsernameNotValid       = errors.New("username is not valid")
	ErrUserServiceKeyNotValid = errors.New("user service key is not valid")
	ErrZoneIdentifierNotSet   = errors.New("zone identifier is not set")
)
