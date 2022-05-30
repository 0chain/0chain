//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
)

// HandleVerifyBlockMessage - handles the verify block message.
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	mc.handleVerifyBlockMessage(ctx, msg)
}

// HandleNotarizationMessage - handles the block notarization message.
func (mc *Chain) HandleNotarizationMessage(ctx context.Context, msg *BlockMessage) {
	mc.handleNotarizationMessage(ctx, msg)
}

// HandleNotarizedBlockMessage - handles a notarized block for a previous round.
func (mc *Chain) HandleNotarizedBlockMessage(ctx context.Context,
	msg *BlockMessage) {
	mc.handleNotarizedBlockMessage(ctx, msg)
}

// HandleVerificationTicketMessage - handles the verification ticket message.
func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	mc.handleVerificationTicketMessage(ctx, msg)
}

// handleVRFShare - handles the vrf share.
func (mc *Chain) HandleVRFShare(ctx context.Context, msg *BlockMessage) {
	handleVRFShare(ctx, msg)
}
