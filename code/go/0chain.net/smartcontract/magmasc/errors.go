package magmasc

import (
	"github.com/0chain/gosdk/zmagmacore/errors"
)

const (
	errCodeBadRequest     = "bad_request"
	errCodeConsumerReg    = "consumer_reg"
	errCodeConsumerUpdate = "consumer_update"
	errCodeDataUsage      = "data_usage"
	errCodeDecode         = "decode_error"
	errCodeFetchData      = "fetch_data"
	errCodeInternal       = "internal_error"
	errCodeProviderReg    = "provider_reg"
	errCodeProviderUpdate = "provider_update"
	errCodeSessionInit    = "session_init"
	errCodeSessionStart   = "session_start"
	errCodeSessionStop    = "session_stop"

	errCodeAccessPointReg    = "access_point_reg"
	errCodeAccessPointUpdate = "access_point_update"

	errCodeRewardPoolLock   = "reward_pool_lock"
	errCodeRewardPoolUnlock = "reward_pool_unlock"
	errCodeTokenPoolCreate  = "token_pool_create"
	errCodeTokenPoolSpend   = "token_pool_spend"

	errTextDecode     = "decode error"
	errTextUnexpected = "unexpected error"

	errCodeInvalidFuncName = "invalid_func_name"
	errTextInvalidFuncName = "function with provided name is not supported"
)

var (
	// errDecodeData represents an error
	// that decode data was failed.
	errDecodeData = errors.New(errCodeDecode, errTextDecode)

	// errInvalidAccessPointID represents an error
	// that access point id was invalidated.
	errInvalidAccessPointID = errors.New(errCodeBadRequest, "invalid access_point_id")

	// errInvalidConsumerExtID represents an error
	// that consumer external id was invalidated.
	errInvalidConsumerExtID = errors.New(errCodeBadRequest, "invalid consumer_ext_id")

	// errInvalidFuncName represents an error that can occur while
	// smart contract is calling with unsupported function name.
	errInvalidFuncName = errors.New(errCodeInvalidFuncName, errTextInvalidFuncName)

	// errInvalidProviderExtID represents an error
	// that provider external id was invalidated.
	errInvalidProviderExtID = errors.New(errCodeBadRequest, "invalid provider_ext_id")

	// errInsufficientFunds represents an error that can occur while
	// check a balance value condition.
	errInsufficientFunds = errors.New(errCodeBadRequest, "insufficient funds")

	// errInternalUnexpected represents an error
	// that internal unexpected issue.
	errInternalUnexpected = errors.New(errCodeInternal, errTextUnexpected)

	// errNegativeValue represents an error that can occur while
	// a checked value is negative.
	errNegativeValue = errors.New(errCodeBadRequest, "negative value")

	// errNilPointerValue represents an error that can occur while
	// a checked value is a nil pointer.
	errNilPointerValue = errors.New(errCodeInternal, "nil pointer value")
)
