package common

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrTemporaryFailure = NewError("temporary_failure", "Please retry again later")

	// ErrNoResource represents error corresponds to http.StatusNotFound.
	ErrNoResource = NewError(ErrNoResourceCode, "can't retrieve resource")

	// ErrBadRequest represents error corresponds to http.StatusBadRequest.
	ErrBadRequest = NewError(ErrBadRequestCode, "request is invalid")

	// ErrInternal represents error corresponds to http.StatusInternalServerError.
	ErrInternal = NewError(ErrInternalCode, "internal server error")

	ErrDecoding = errors.New("decoding error")

	ErrNotModified = errors.New("not modified")
)

const (
	ErrNoResourceCode = "resource_not_found"
	ErrBadRequestCode = "invalid_request"
	ErrInternalCode   = "internal_error"
)

/*Error type for a new application error */
type Error struct {
	Code string `json:"code,omitempty"`
	Msg  string `json:"msg"`
}

func (err *Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Code, err.Msg)
}

func (err *Error) Is(target error) bool {
	t, ok := target.(*Error)
	if ok {
		return err.Code == t.Code
	}

	return false
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

// NewErrInternal creates new Error with ErrInternalCode.
func NewErrInternal(msgs ...string) error {
	if len(msgs) == 0 {
		return ErrNoResource
	}

	return NewError(ErrInternalCode, strings.Join(msgs, ": "))
}

// NewErrNoResource creates new Error with ErrNoResourceCode.
func NewErrNoResource(msgs ...string) error {
	if len(msgs) == 0 {
		return ErrNoResource
	}

	return NewError(ErrNoResourceCode, strings.Join(msgs, ": "))
}

// NewErrBadRequest creates new Error with ErrBadRequestCode.
func NewErrBadRequest(msgs ...string) error {
	if len(msgs) == 0 {
		return ErrBadRequest
	}

	return NewError(ErrBadRequestCode, strings.Join(msgs, ": "))
}
