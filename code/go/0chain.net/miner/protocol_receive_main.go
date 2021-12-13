//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
)

func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	mc.handleVerificationTicketMessage(ctx, msg)
}
