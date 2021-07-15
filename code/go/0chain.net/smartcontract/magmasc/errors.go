package magmasc

import (
	"errors"
	"sync"
)

const (
	errDelim = ": "

	errCodeAcceptTerms    = "accept_terms"
	errCodeBadRequest     = "bad_request"
	errCodeConsumerReg    = "consumer_reg"
	errCodeConsumerUpdate = "consumer_update"
	errCodeDataUsage      = "data_usage"
	errCodeDecode         = "decode_error"
	errCodeFetchData      = "fetch_data"
	errCodeInternal       = "internal_error"
	errCodeProviderReg    = "provider_reg"
	errCodeProviderUpdate = "provider_update"
	errCodeSessionStop    = "session_stop"

	errCodeTokenPoolCreate = "token_pool_create"
	errCodeTokenPoolSpend  = "token_pool_spend"

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
		rwmx sync.RWMutex
	}
)

var (
	// errDecodeData represents an error
	// that decode data was failed.
	errDecodeData = errNew(errCodeDecode, errTextDecode)

	// errInvalidAccessPointID represents an error
	// that access point id was invalidated.
	errInvalidAccessPointID = errNew(errCodeBadRequest, "invalid access_point_id")

	// errInvalidAcknowledgment represents an error
	// that an acknowledgment was invalidated.
	errInvalidAcknowledgment = errNew(errCodeInternal, errTextAcknInvalid)

	// errInvalidConsumer represents an error
	// that consumer was invalidated.
	errInvalidConsumer = errNew(errCodeInternal, "invalid consumer")

	// errInvalidConsumerExtID represents an error
	// that consumer external id was invalidated.

	errInvalidConsumerExtID = errNew(errCodeBadRequest, "invalid consumer_ext_id")
	// errInvalidDataUsage represents an error
	// that a data usage was invalidated.
	errInvalidDataUsage = errNew(errCodeInternal, "invalid data usage")

	// errInvalidFuncName represents an error that can occur while
	// smart contract is calling with unsupported function name.
	errInvalidFuncName = errNew(errCodeInvalidFuncName, errTextInvalidFuncName)

	// errInvalidProvider represents an error
	// that provider was invalidated.
	errInvalidProvider = errNew(errCodeInternal, "invalid provider")

	// errInvalidProviderExtID represents an error
	// that provider external id was invalidated.
	errInvalidProviderExtID = errNew(errCodeBadRequest, "invalid provider_ext_id")

	// errInvalidProviderTerms represents an error
	// that provider terms was invalidated.
	errInvalidProviderTerms = errNew(errCodeInternal, "invalid provider terms")

	// errInsufficientFunds represents an error that can occur while
	// check a balance value condition.
	errInsufficientFunds = errNew(errCodeBadRequest, "insufficient funds")

	// errInternalUnexpected represents an error
	// that internal unexpected issue.
	errInternalUnexpected = errNew(errCodeInternal, errTextUnexpected)

	// errNegativeValue represents an error that can occur while
	// a checked value is negative.
	errNegativeValue = errNew(errCodeBadRequest, "negative value")
)

// Error implements error interface.
func (m *errWrapper) Error() string {
	m.rwmx.RLock()
	defer m.rwmx.RUnlock()

	return m.code + errDelim + m.text
}

// Unwrap implements error unwrap interface.
func (m *errWrapper) Unwrap() error {
	return m.wrap
}

// WrapErr implements error wrapper interface.
func (m *errWrapper) WrapErr(err error) *errWrapper {
	m.rwmx.Lock()
	defer m.rwmx.Unlock()

	if err != nil && !errors.Is(m, err) {
		m.wrap = err
		m.text += errDelim + err.Error()
	}

	return m
}

// errAny reports whether an error in error's chain
// matches to any error provided in list.
func errAny(err error, targets ...error) bool {
	for _, target := range targets {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
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
	wrapper := &errWrapper{code: code, text: text}
	if err != nil && !errors.Is(wrapper, err) {
		return wrapper.WrapErr(err)
	}

	return wrapper
}
