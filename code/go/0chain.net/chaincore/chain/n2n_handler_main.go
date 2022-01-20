//go:build !integration_tests
// +build !integration_tests

package chain

func SetupX2MRequestors() {
	setupX2MRequestors()
}

func (c *Chain) BlockStateChangeHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	return c.blockStateChangeHandler(ctx, r)
}
