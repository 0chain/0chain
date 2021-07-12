package magmasc

import (
	"errors"
)

const (
	errDelim = ": "

	errCodeAcceptTerms    = "accept_terms"
	errCodeBadRequest     = "bad_request"
	errCodeConsumerReg    = "consumer_reg"
	errCodeDataUsage      = "data_usage"
	errCodeDecode         = "decode_error"
	errCodeFetchData      = "fetch_data"
	errCodeInternal       = "internal_error"
	errCodeProviderReg    = "provider_reg"
	errCodeProviderUpdate = "provider_update"
	errCodeSessionStop    = "session_stop"
	errCodeUpdateData     = "update_data"

	errCodeTokenPoolCreate   = "token_pool_create"
	errCodeTokenPoolSpend    = "token_pool_spend"
	errCodeTokenPoolTransfer = "token_pool_transfer"

	errTextAcknInvalid = "acknowledgment invalid"
	errTextDecode      = "decode error"
	errTextUnexpected  = "unexpected error"

	errCodeInvalidFuncName = "invalid_func_name"
	errTextInvalidFuncName = "function with provided name is not supported"
)

type (
	// wrapper implements Wrapper interface.
	errWrapper struct {
		code string
		text string
		wrap error
	}
)

var (
	// errAcknowledgmentInvalid represents an error
	// that an acknowledgment was invalidated.
	errAcknowledgmentInvalid = errNew(errCodeInternal, errTextAcknInvalid)

	// errDataUsageInvalid represents an error
	// that a data usage was invalidated.
	errDataUsageInvalid = errNew(errCodeInternal, "data usage invalid")

	// errDecodeData represents an error
	// that decode data was failed.
	errDecodeData = errNew(errCodeDecode, errTextDecode)

	// errConsumerAlreadyExists represents an error that can occur while
	// Consumer is creating and saving in blockchain state.
	errConsumerAlreadyExists = errNew(errCodeInternal, "consumer already exists")

	// errInvalidFuncName represents an error that can occur while
	// smart contract is calling with unsupported function name.
	errInvalidFuncName = errNew(errCodeInvalidFuncName, errTextInvalidFuncName)

	// errInsufficientFunds represents an error that can occur while
	// check a balance value condition.
	errInsufficientFunds = errNew(errCodeBadRequest, "insufficient funds")

	// errNegativeValue represents an error that can occur while
	// a checked value is negative.
	errNegativeValue = errNew(errCodeBadRequest, "negative value")

	// errProviderAlreadyExists represents an error that can occur while
	// Provider is creating and saving in blockchain state.
	errProviderAlreadyExists = errNew(errCodeInternal, "provider already exists")

	// errProviderTermsInvalid represents an error
	// that provider terms was invalidated.
	errProviderTermsInvalid = errNew(errCodeInternal, "provider terms invalid")

	// errVerifyAccessPoint represents an error
	// that verify access point id failed.
	errVerifyAccessPointID = errNew(errCodeBadRequest, "verify access point id failed")

	// errVerifyAccessPoint represents an error
	// that verify consumer id failed.
	errVerifyConsumerID = errNew(errCodeBadRequest, "verify consumer id failed")

	// errVerifyAccessPoint represents an error
	// that verify provider id failed.
	errVerifyProviderID = errNew(errCodeBadRequest, "verify provider id failed")
)

// Error implements error interface.
func (m *errWrapper) Error() string {
	return m.code + errDelim + m.text
}

// Unwrap implements error unwrap interface.
func (m *errWrapper) Unwrap() error {
	return m.wrap
}

// WrapErr implements error wrapper interface.
func (m *errWrapper) WrapErr(err error) *errWrapper {
	if err != nil && !errors.Is(m, err) {
		m.wrap = err
		m.text += errDelim + err.Error()
	}

	return m
}

// errIs wraps function errors.Is from stdlib to avoid import it
// in other places of the magma smart contract (magmasc) package.
func errIs(err, target error) bool {
	return errors.Is(err, target)
}

// errNew returns constructed error wrapper interface.
func errNew(code, text string) *errWrapper {
	return &errWrapper{code: code, text: text}
}

// errWrap wraps given error into a new error with format.
func errWrap(code, text string, err error) *errWrapper {
	wrapper := errWrapper{code: code, text: text}
	if err != nil && !errors.Is(&wrapper, err) {
		wrapper.wrap = err
		wrapper.text += errDelim + err.Error()
	}

	return &wrapper
}
