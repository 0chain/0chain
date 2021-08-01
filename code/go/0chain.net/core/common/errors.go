package common

import (
	"fmt"
	"strings"

	zchainErrors "github.com/0chain/gosdk/errors"
	"github.com/pkg/errors"
)

var (
	ErrTemporaryFailure = zchainErrors.New("temporary_failure", "Please retry again later")

	// ErrNoResource represents error corresponds to http.StatusNotFound.
	ErrNoResource = zchainErrors.New(ErrNoResourceCode, "can't retrieve resource")

	// ErrBadRequest represents error corresponds to http.StatusBadRequest.
	ErrBadRequest = zchainErrors.New(ErrBadRequestCode, "request is invalid")

	// ErrInternal represents error corresponds to http.StatusInternalServerError.
	ErrInternal = zchainErrors.New(ErrInternalCode, "internal server error")

	// ErrDecoding represents error corresponds to common decoding error
	ErrDecoding = zchainErrors.New("decoding error")
)

const (
	ErrNoResourceCode = "resource_not_found"
	ErrBadRequestCode = "invalid_request"
	ErrInternalCode   = "internal_error"
)

/*InvalidRequest - create error messages that are needed when validating request input */
func InvalidRequest(msg string) error {
	return zchainErrors.New("invalid_request", fmt.Sprintf("Invalid request (%v)", msg))
}

// NewErrInternal creates new Error with ErrInternalCode.
func NewErrInternal(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrNoResource
	}

	return errors.Wrap(err, zchainErrors.New(ErrInternalCode, strings.Join(msgs, ": ")).Error())
}

// NewErrNoResource creates new Error with ErrNoResourceCode.
func NewErrNoResource(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrNoResource
	}

	return errors.Wrap(err, zchainErrors.New(ErrNoResourceCode, strings.Join(msgs, ": ")).Error())
}

// NewErrBadRequest creates new Error with ErrBadRequestCode.
func NewErrBadRequest(err error, msgs ...string) error {
	if len(msgs) == 0 && err == nil {
		return ErrBadRequest
	}

	return errors.Wrap(err, zchainErrors.New(ErrBadRequestCode, strings.Join(msgs, ": ")).Error())
}
