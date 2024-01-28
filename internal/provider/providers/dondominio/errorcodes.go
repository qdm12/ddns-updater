package dondominio

import (
	"fmt"

	"github.com/qdm12/ddns-updater/internal/provider/errors"
)

func makeError(errorCode int, errorMessage string) error {
	switch errorCode {
	case syntaxError, syntaxErrorParameterFault, invalidObjectOrAction,
		notImplementedObjectOrAction, syntaxErrorInvalidParameter, accountDeleted,
		accountNotExists, invalidDomainName, tldNotSupported:
		return fmt.Errorf("%w: %s (%d)", errors.ErrBadRequest, errorMessage, errorCode)
	case notAllowedObjectOrAction, loginRequired, loginInvalid, sessionInvalid,
		actionNotAllowed, accountInvalidPass1, accountInvalidPass2, accountInvalidPass3,
		domainUpdateNotAllowed:
		return fmt.Errorf("%w: %s (%d)", errors.ErrAuth, errorMessage, errorCode)
	case accountBlocked1, accountBlocked2, accountBlocked3, accountBlocked4,
		accountBlocked5, accountBlocked6, accountFiltered1, accountFiltered2,
		accountBanned, domainUpdateBlocked:
		return fmt.Errorf("%w: %s (%d)", errors.ErrBannedAbuse, errorMessage, errorCode)
	case accountInactive:
		return fmt.Errorf("%w: %s (%d)", errors.ErrAccountInactive, errorMessage, errorCode)
	case domainCheckError, domainNotFound:
		return fmt.Errorf("%w: %s (%d)", errors.ErrDomainNotFound, errorMessage, errorCode)
	default:
		return fmt.Errorf("%w: %s (%d)", errors.ErrUnknownResponse, errorMessage, errorCode)
	}
}

// See section "10.1.1 Error codes" at https://dondominio.dev/en/api/docs/api/#tables
const (
	success                                                    = 0
	undefinedError                                             = 1
	syntaxError                                                = 100
	syntaxErrorParameterFault                                  = 101
	invalidObjectOrAction                                      = 102
	notAllowedObjectOrAction                                   = 103
	notImplementedObjectOrAction                               = 104
	syntaxErrorInvalidParameter                                = 105
	loginRequired                                              = 200
	loginInvalid                                               = 201
	sessionInvalid                                             = 210
	actionNotAllowed                                           = 300
	accountBlocked1                                            = 1000
	accountDeleted                                             = 1001
	accountInactive                                            = 1002
	accountNotExists                                           = 1003
	accountInvalidPass1                                        = 1004
	accountInvalidPass2                                        = 1005
	accountBlocked2                                            = 1006
	accountFiltered1                                           = 1007
	accountInvalidPass3                                        = 1009
	accountBlocked3                                            = 1010
	accountBlocked4                                            = 1011
	accountBlocked5                                            = 1012
	accountBlocked6                                            = 1013
	accountFiltered2                                           = 1014
	accountBanned                                              = 1030
	insufficientBalance                                        = 1100
	invalidDomainName                                          = 2001
	tldNotSupported                                            = 2002
	tldInMaintenance                                           = 2003
	domainCheckError                                           = 2004
	domainTransferNotAllowed                                   = 2005
	domainWhoisNotAllowed                                      = 2006
	domainWhoisError                                           = 2007
	domainNotFound                                             = 2008
	domainCreateError                                          = 2009
	domainCreateErrorTaken                                     = 2010
	domainCreateErrorDomainPremium                             = 2011
	domainTransferError                                        = 2012
	domainRenewError                                           = 2100
	domainRenewNotAllowed                                      = 2101
	domainRenewBlocked                                         = 2102
	domainUpdateError                                          = 2200
	domainUpdateNotAllowed                                     = 2201
	domainUpdateBlocked                                        = 2202
	invalidOperationDueToTheOwnerContactDataVerificationStatus = 2210
	contactNotExists                                           = 3001
	contactDataError                                           = 3002
	invalidOperationDueToTheContactDataVerification            = 3003
	userNotExists                                              = 3500
	userCreateError                                            = 3501
	userUpdateError                                            = 3502
	serviceNotFound                                            = 4001
	serviceEntityNotFound                                      = 4002
	maximumAmountOfEntitiesError                               = 4003
	failureToCreateTheEntity                                   = 4004
	failureToUpdateTheEntity                                   = 4005
	failureToDeleteTheEntity                                   = 4006
	failureToCreateTheService                                  = 4007
	failureToUpgradeTheService                                 = 4008
	failureToRenewTheService                                   = 4009
	failureToMotifyTheParkingSystem                            = 4010
	sslError                                                   = 5000
	sslNotFound                                                = 5001
	webConstructorError                                        = 10001
)
