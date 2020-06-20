// +build !integration_tests

package miner

import (
	"context"
)

// average wrapper
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	return handleVerifyBlockMessage(ctx, msg)
}
