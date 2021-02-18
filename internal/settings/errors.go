package settings

import "errors"

var (
	ErrIPv6NotSupported = errors.New("IPv6 is not supported by this provider")
)

// Validation errors.
var (
	ErrEmptyName               = errors.New("empty name")
	ErrEmptyPassword           = errors.New("empty password")
	ErrEmptyKey                = errors.New("empty key")
	ErrEmptyAppKey             = errors.New("empty app key")
	ErrEmptyConsumerKey        = errors.New("empty consumer key")
	ErrEmptySecret             = errors.New("empty secret")
	ErrEmptyToken              = errors.New("empty token")
	ErrEmptyTTL                = errors.New("TTL is not set")
	ErrEmptyUsername           = errors.New("empty username")
	ErrEmptyZoneIdentifier     = errors.New("empty zone identifier")
	ErrHostOnlyAt              = errors.New(`host can only be "@"`)
	ErrHostOnlySubdomain       = errors.New("host can only be a subdomain")
	ErrHostWildcard            = errors.New(`host cannot be a "*"`)
	ErrMalformedEmail          = errors.New("malformed email address")
	ErrMalformedKey            = errors.New("malformed key")
	ErrMalformedPassword       = errors.New("malformed password")
	ErrMalformedToken          = errors.New("malformed token")
	ErrMalformedUsername       = errors.New("malformed username")
	ErrMalformedUserServiceKey = errors.New("malformed user service key")
)

// Intermediary steps errors.
var (
	ErrCreateRecord    = errors.New("cannot create record")
	ErrGetDomainID     = errors.New("cannot get domain ID")
	ErrGetRecordID     = errors.New("cannot get record ID")
	ErrGetRecordInZone = errors.New("cannot get record in zone") // LuaDNS
	ErrGetZoneID       = errors.New("cannot get zone ID")        // LuaDNS
	ErrListRecords     = errors.New("cannot list records")       // Dreamhost
	ErrRemoveRecord    = errors.New("cannot remove record")      // Dreamhost
	ErrUpdateRecord    = errors.New("cannot update record")
)

// Update errors.
var (
	ErrAbuse                   = errors.New("banned due to abuse")
	ErrAccountInactive         = errors.New("account is inactive")
	ErrAuth                    = errors.New("bad authentication")
	ErrRequestEncode           = errors.New("cannot encode request")
	ErrBadHTTPStatus           = errors.New("bad HTTP status")
	ErrBadRequest              = errors.New("bad request sent")
	ErrBannedUserAgent         = errors.New("user agend is banned")
	ErrConflictingRecord       = errors.New("conflicting record")
	ErrDNSServerSide           = errors.New("server side DNS error")
	ErrDomainDisabled          = errors.New("record disabled")
	ErrDomainIDNotFound        = errors.New("ID not found in domain record")
	ErrFeatureUnavailable      = errors.New("feature is not available to the user")
	ErrHostnameNotExists       = errors.New("hostname does not exist")
	ErrInvalidSystemParam      = errors.New("invalid system parameter")
	ErrIPReceivedMalformed     = errors.New("malformed IP address received")
	ErrIPReceivedMismatch      = errors.New("mismatching IP address received")
	ErrMalformedIPSent         = errors.New("malformed IP address sent")
	ErrNoResultReceived        = errors.New("no result received")
	ErrNotFound                = errors.New("not found")
	ErrNumberOfResultsReceived = errors.New("wrong number of results received")
	ErrPrivateIPSent           = errors.New("private IP cannot be routed")
	ErrRecordNotEditable       = errors.New("record is not editable") // Dreamhost
	ErrRecordNotFound          = errors.New("record not found")
	ErrRequestMarshal          = errors.New("cannot marshal request body")
	ErrUnknownResponse         = errors.New("unknown response received")
	ErrUnmarshalResponse       = errors.New("cannot unmarshal update response")
	ErrUnsuccessfulResponse    = errors.New("unsuccessful response")
	ErrZoneNotFound            = errors.New("zone not found") // LuaDNS
)
