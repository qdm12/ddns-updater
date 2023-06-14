package errors

import "errors"

var (
	ErrCreateRecord    = errors.New("cannot create record")
	ErrGetDomainID     = errors.New("cannot get domain ID")
	ErrGetRecordID     = errors.New("cannot get record ID")
	ErrGetRecordInZone = errors.New("cannot get record in zone") // LuaDNS
	ErrGetZoneID       = errors.New("cannot get zone ID")        // LuaDNS
	ErrListRecords     = errors.New("cannot list records")       // Dreamhost
	ErrRemoveRecord    = errors.New("cannot remove record")      // Dreamhost
	ErrUpdateRecord    = errors.New("cannot update record")
	ErrSessionIsEmpty  = errors.New("session received is empty") // Netcup
)
