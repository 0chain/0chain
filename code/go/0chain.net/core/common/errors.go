package common

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrTemporaryFailure = NewError("temporary_failure", "Please retry again later")

	// ErrNoResource represents error corresponds to http.StatusNotFound.
	ErrNoResource = NewError("resource_not_found", "can't retrieve resource")

	// ErrBadRequest represents error corresponds to http.StatusBadRequest.
	ErrBadRequest = NewError("bad_request", "request is invalid")

	// ErrInternal represents error corresponds to http.StatusInternalServerError.
	ErrInternal = NewError("internal", "internal server error")

	ErrDecoding = errors.New("decoding error")
)

/*Error type for a new application error */
type Error struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg"`
}

func (err *Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Code, err.Msg)
}

/*NewError - create a new error */
func NewError(code string, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

/*NewErrorf - create a new formated error */
func NewErrorf(code string, format string, args ...interface{}) *Error {
	return &Error{Code: code, Msg: fmt.Sprintf(format, args...)}
}

/*InvalidRequest - create error messages that are needed when validating request input */
func InvalidRequest(msg string) error {
	return NewError("invalid_request", fmt.Sprintf("Invalid request (%v)", msg))
}

// WrapErrInternal wraps ErrInternal in new error with provided messages and returns resulted error.
func WrapErrInternal(msgs ...string) error {
	if len(msgs) == 0 {
		return ErrNoResource
	}

	return fmt.Errorf("%w: %s", ErrInternal, strings.Join(msgs, ": "))
}

// WrapErrNoResource wraps ErrNoResource in new error with provided messages and returns resulted error.
func WrapErrNoResource(msgs ...string) error {
	if len(msgs) == 0 {
		return ErrNoResource
	}

	return fmt.Errorf("%w: %s", ErrNoResource, strings.Join(msgs, ": "))
}

// WrapErrBadRequest wraps ErrBadRequest in new error with provided messages and returns resulted error.
func WrapErrBadRequest(msgs ...string) error {
	if len(msgs) == 0 {
		return ErrBadRequest
	}

	return fmt.Errorf("%w: %s", ErrBadRequest, strings.Join(msgs, ": "))
}
