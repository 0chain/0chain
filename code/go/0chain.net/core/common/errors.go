package common

import (
	"fmt"
	"strings"

	"github.com/0chain/gosdk/core/common/errors"
)

var (
	ErrTemporaryFailure = errors.Register("temporary_failure", "Please retry again later")

	// ErrNoResource represents error corresponds to http.StatusNotFound.
	ErrNoResource = errors.Register(ErrNoResourceCode, "can't retrieve resource")

	// ErrBadRequest represents error corresponds to http.StatusBadRequest.
	ErrBadRequest = errors.Register(ErrBadRequestCode, "request is invalid")

	// ErrInternal represents error corresponds to http.StatusInternalServerError.
	ErrInternal = errors.Register(ErrInternalCode, "internal server error")

	// ErrDecoding represents error corresponds to common decoding error
	ErrDecoding = errors.Register("decoding error")
)

const (
	ErrNoResourceCode = "resource_not_found"
	ErrBadRequestCode = "invalid_request"
	ErrInternalCode   = "internal_error"
)

/*InvalidRequest - create error messages that are needed when validating request input */
func InvalidRequest(msg string) error {
	return errors.New("invalid_request", fmt.Sprintf("Invalid request (%v)", msg))
}

// NewErrInternal creates new Error with ErrInternalCode.
func NewErrInternal(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrNoResource()
	}

	return errors.Wrap(err, errors.New(ErrInternalCode, strings.Join(msgs, ": ")))
}

// NewErrNoResource creates new Error with ErrNoResourceCode.
func NewErrNoResource(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrNoResource()
	}

	return errors.Wrap(err, errors.New(ErrNoResourceCode, strings.Join(msgs, ": ")))
}

// NewErrBadRequest creates new Error with ErrBadRequestCode.
func NewErrBadRequest(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrBadRequest()
	}

	return errors.Wrap(err, errors.New(ErrBadRequestCode, strings.Join(msgs, ": ")))
}
