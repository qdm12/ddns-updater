package errors

import "errors"

var (
	ErrAbuse                   = errors.New("banned due to abuse")
	ErrAccountInactive         = errors.New("account is inactive")
	ErrAuth                    = errors.New("bad authentication")
	ErrBadHTTPStatus           = errors.New("bad HTTP status")
	ErrBadRequest              = errors.New("bad request sent")
	ErrBannedUserAgent         = errors.New("user agend is banned")
	ErrConflictingRecord       = errors.New("conflicting record")
	ErrDNSServerSide           = errors.New("server side DNS error")
	ErrDomainDisabled          = errors.New("record disabled")
	ErrDomainNotFound          = errors.New("domain not found")
	ErrDomainIDNotFound        = errors.New("ID not found in domain record")
	ErrFeatureUnavailable      = errors.New("feature is not available to the user")
	ErrHostnameNotExists       = errors.New("hostname does not exist")
	ErrInvalidSystemParam      = errors.New("invalid system parameter")
	ErrIPReceivedMalformed     = errors.New("malformed IP address received")
	ErrIPReceivedMismatch      = errors.New("mismatching IP address received")
	ErrMalformedIPSent         = errors.New("malformed IP address sent")
	ErrNoIPInResponse          = errors.New("no IP address in response")
	ErrNoResultReceived        = errors.New("no result received")
	ErrNotFound                = errors.New("not found")
	ErrNoService               = errors.New("no service")
	ErrNumberOfResultsReceived = errors.New("wrong number of results received")
	ErrPrivateIPSent           = errors.New("private IP cannot be routed")
	ErrRecordNotEditable       = errors.New("record is not editable") // Dreamhost
	ErrRecordNotFound          = errors.New("record not found")
	ErrRequestEncode           = errors.New("cannot encode request")
	ErrRequestMarshal          = errors.New("cannot marshal request body")
	ErrUnknownResponse         = errors.New("unknown response received")
	ErrUnmarshalResponse       = errors.New("cannot unmarshal update response")
	ErrUnsuccessfulResponse    = errors.New("unsuccessful response")
	ErrZoneNotFound            = errors.New("zone not found") // LuaDNS
)
