package smartcontract

import (
	"fmt"
	"strings"

	"0chain.net/core/common"
	"0chain.net/core/util"
	"github.com/0chain/gosdk/core/common/errors"
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
		return common.NewErrInternal(err, msgs...)
	case errors.Is(err, util.ErrValueNotPresent), errors.Is(err, util.ErrNodeNotFound):
		return common.NewErrNoResource(err, msgs...)
	default:
		if defaultInternal {
			return common.NewErrInternal(err, msgs...)
		}

		if len(msgs) == 0 {
			return err
		}

		return errors.Wrap(err, fmt.Sprintf("%s: ", strings.Join(msgs, ":")))
	}
}
