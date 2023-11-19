//go:build !integration_tests
// +build !integration_tests

package chain

import (
	"context"

	"0chain.net/chaincore/block"
)

func (c *Chain) postFinalize(ctx context.Context, fb *block.Block) error {
	return nil
}
