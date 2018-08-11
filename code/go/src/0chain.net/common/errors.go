package common

import (
	"fmt"
)

var ErrTemporaryFailure = NewError("temporary_failure", "Please retry again later")

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

/*InvalidRequest - create error messages that are needed when validating request input */
func InvalidRequest(msg string) error {
	return NewError("invalid_request", fmt.Sprintf("Invalid request (%v)", msg))
}
