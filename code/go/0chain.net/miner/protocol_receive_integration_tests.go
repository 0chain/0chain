// +build integration_tests

package miner

import (
	"context"

	crpc "0chain.net/conductor/conductrpc"
)

// average wrapper
func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {

	var (
		state = crpc.Client().State()
		b     = msg.Block
	)

	if !state.SignOnlyCompetingBlocks.IsCompetingGroupMember(state, b.MinerID) {
		return // don't verify the block, drop it
	}

	return handleVerifyBlockMessage(ctx, msg)
}
