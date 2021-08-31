package common

import (
	"strings"

	"github.com/0chain/errors"
)

var (
	ErrTemporaryFailure = errors.New("temporary_failure", "Please retry again later")

	// ErrNoResource represents error corresponds to http.StatusNotFound.
	ErrNoResource = errors.New(ErrNoResourceCode, "can't retrieve resource")

	// ErrBadRequest represents error corresponds to http.StatusBadRequest.
	ErrBadRequest = errors.New(ErrBadRequestCode, "request is invalid")

	// ErrInternal represents error corresponds to http.StatusInternalServerError.
	ErrInternal = errors.New(ErrInternalCode, "internal server error")

	// ErrDecoding represents error corresponds to common decoding error
	ErrDecoding = errors.New("", "decoding error")
)

const (
	ErrNoResourceCode = "resource_not_found"
	ErrBadRequestCode = "invalid_request"
	ErrInternalCode   = "internal_error"
)

/*InvalidRequest - create error messages that are needed when validating request input */
func InvalidRequest(msg string) error {
	return errors.Newf("invalid_request", "Invalid request (%v)", msg)
}

// NewErrInternal creates new Error with ErrInternalCode.
func NewErrInternal(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrNoResource
	}

	if err == nil {
		return errors.New(ErrInternalCode, strings.Join(msgs, ": "))
	}
	return errors.Wrap(err, errors.New(ErrInternalCode, strings.Join(msgs, ": ")).Error())
}

// NewErrNoResource creates new Error with ErrNoResourceCode.
func NewErrNoResource(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrNoResource
	}
	if err == nil {
		return errors.New(ErrNoResourceCode, strings.Join(msgs, ": "))
	}

	return errors.Wrap(err, errors.New(ErrNoResourceCode, strings.Join(msgs, ": ")).Error())
}

// NewErrBadRequest creates new Error with ErrBadRequestCode.
func NewErrBadRequest(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrBadRequest
	}

	if err == nil {
		return errors.New(ErrBadRequestCode, strings.Join(msgs, ": "))
	}

	return errors.Wrap(err, errors.New(ErrBadRequestCode, strings.Join(msgs, ": ")).Error())
}
