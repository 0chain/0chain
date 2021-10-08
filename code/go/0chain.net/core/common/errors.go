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
)

const (
	ErrNoResourceCode = "resource_not_found"
	ErrBadRequestCode = "invalid_request"
	ErrInternalCode   = "internal_error"

	// GRPC Code "Internal (13)"
	Internal CodeString = "Internal"
	// GRPC Code "Not Found (5)"
	Notfound CodeString = "NotFound"
	// InvalidGRPCRequest is HTTP Status "400 Bad Request" and GRPC Code "400 Bad Request".
	InvalidGRPCRequest CodeString = "InvalidRequest"
	// Unauthenticated
	Unauthenticated CodeString = "Unauthenticated"
	// PermissionDenied
	PermissionDenied CodeString = "PermissionDenied"
	// TemporaryUnavailable
	TemporaryUnavailable CodeString = "TemporaryUnavailable"
	// Canceled
	Canceled CodeString = "Canceled"
	// Timeout
	Timeout CodeString = "Timeout"
	// Unknown
	Unknown CodeString = "Unknown"
)

type Code interface {
	ErrorCode() string
}

// StringCode represents an error Code in string.
type CodeString string

// ErrorCode implements the Code interface so that
func (c CodeString) ErrorCode() string {
	return string(c)
}

/*Error type for a new application error */
type Error struct {
	Code CodeString `json:"code,omitempty"`
	Msg  string     `json:"msg"`
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
func NewError(code CodeString, msg string) *Error {
	return &Error{Code: code, Msg: msg}
}

/*NewErrorf - create a new formated error */
func NewErrorf(code CodeString, format string, args ...interface{}) *Error {
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
