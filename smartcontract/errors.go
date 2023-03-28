package smartcontract

import (
	"errors"
	"fmt"
	"strings"

	"0chain.net/core/common"
	"github.com/0chain/common/core/util"
)

// NewErrNoResourceOrErrInternal wraps err, passed in args, in common.ErrInternal or in common.ErrNoResource, depending on
// passed error.
//
// Supported no resource errors:
//
// util.ErrValueNotPresent, util.ErrNodeNotFound.
//
// Supported internal errors:
//
// common.ErrDecoding.
//
// If defaultInternal is true and provided error isn't supported,
// NewErrNoResourceOrErrInternal will wrap provided error in common.ErrInternal,
// If value isn't supported and defaultInternal is false,
// NewErrNoResourceOrErrInternal returns provided error without wrapping.
func NewErrNoResourceOrErrInternal(err error, defaultInternal bool, msgs ...string) error {
	switch {
	case errors.Is(err, common.ErrDecoding):
		return common.NewErrInternal(strings.Join(msgs, ": "), err.Error())
	case errors.Is(err, util.ErrValueNotPresent), errors.Is(err, util.ErrNodeNotFound):
		return common.NewErrNoResource(strings.Join(msgs, ": "), err.Error())
	default:
		if defaultInternal {
			return common.NewErrInternal(strings.Join(msgs, ": "), err.Error())
		}

		if len(msgs) == 0 {
			return err
		}

		return fmt.Errorf("%s: %w", strings.Join(msgs, ": "), err)
	}
}
