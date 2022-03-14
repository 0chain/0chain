//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"
	"net/http"
)

func SetupX2MRequestors() {
	setupX2MRequestors()
}

func SetupX2SRequestors() {
	setupX2SRequestors()
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return c.blockStateChangeHandler(ctx, r)
}
