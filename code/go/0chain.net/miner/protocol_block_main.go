//go:build !integration_tests
// +build !integration_tests

package miner

import (
	"context"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"go.uber.org/zap"
)

func (mc *Chain) SignBlock(ctx context.Context, b *block.Block) (
	bvt *block.BlockVerificationTicket, err error) {

	return mc.signBlock(ctx, b)
}

// add hash to generated block and sign it
func (mc *Chain) hashAndSignGeneratedBlock(ctx context.Context,
	b *block.Block) (err error) {

	var self = node.Self
	b.HashBlock()
	b.Signature, err = self.Sign(b.Hash)
	return
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) error {
	go func() {
		mc.updateFinalizedBlock(ctx, b) // nolint: errcheck
	}()

	fr := mc.GetRound(b.Round)
	if fr != nil {
		fr.Finalize(b)
	}

	// return if view change is off
	if !mc.IsViewChangeEnabled() {
		return nil
	}

	// perform view change (or not perform)
	if err := mc.ViewChange(ctx, b); err != nil {
		logging.Logger.Error("[mvc] view change", zap.Int64("round", b.Round), zap.Error(err))
		return err
	}

	nodeLists, err := mc.GetRegisterNodesList(b)
	if err != nil {
		logging.Logger.Debug("update finalized block - get node lists failed", zap.Error(err))
	} else {
		// update the register node list cache
		node.UpdateVCAddNodesCache(nodeLists)
	}

	pn, err := mc.GetPhaseOfBlock(b)
	if err != nil && err != util.ErrValueNotPresent {
		// Non-fatal. Missing MPT nodes should not block finalization.
		logging.Logger.Warn("update finalized block - get phase of block failed (non-fatal)",
			zap.Int64("round", b.Round), zap.Error(err))
		return nil
	}

	if pn == nil {
		return nil
	}

	logging.Logger.Debug("[mvc] update finalized block - send phase node",
		zap.Int64("round", b.Round),
		zap.Int64("start_round", pn.StartRound),
		zap.String("phase", pn.Phase.String()))
	go mc.SendPhaseNode(context.Background(), chain.PhaseEvent{Phase: *pn})
	return nil
}

func (mc *Chain) GenerateBlock(ctx context.Context,
	b *block.Block,
	waitOver bool,
	waitC chan struct{}) error {
	return mc.generateBlockWorker.Run(ctx, func() error {
		err := mc.generateBlock(ctx, b, minerChain, waitOver, waitC)
		if err == nil {
			return nil
		}

		// If generation failed due to corrupted or missing state, try to
		// recover and retry once.
		if !cstate.ErrInvalidState(err) {
			return err
		}

		logging.Logger.Warn("generate block - state error, recovering",
			zap.Int64("round", b.Round),
			zap.Error(err))

		if b.PrevBlock == nil {
			return err
		}

		// Collect the specific missing node keys from the block state.
		// The block's client state is the MPT that encountered the missing
		// nodes during traversal (set early in generateBlock).
		var missingKeys []util.Key
		if b.ClientState != nil {
			missingKeys = b.ClientState.GetMissingNodeKeys()
		}

		if len(missingKeys) > 0 {
			// Fetch the specific missing nodes from peers (sharders have
			// the full persisted state for finalized blocks).
			logging.Logger.Info("generate block - syncing missing state nodes from peers",
				zap.Int64("round", b.Round),
				zap.Int("missing_keys", len(missingKeys)))
			syncCtx, syncCancel := context.WithTimeout(ctx, 10*time.Second)
			syncErr := mc.GetStateNodes(syncCtx, missingKeys)
			syncCancel()
			if syncErr != nil {
				logging.Logger.Error("generate block - failed to sync missing nodes",
					zap.Int64("round", b.Round), zap.Error(syncErr))
				return err
			}
			logging.Logger.Info("generate block - missing nodes synced, retrying",
				zap.Int64("round", b.Round))
		} else {
			// No specific missing keys tracked — cannot recover without
			// knowing which nodes are missing. Return the error and let
			// the round timeout naturally. Do NOT mutate b.PrevBlock as
			// it is a shared pointer in the block cache.
			logging.Logger.Warn("generate block - state error with no tracked missing keys, cannot recover",
				zap.Int64("round", b.Round),
				zap.String("prev_block", b.PrevHash),
				zap.String("prev_state", util.ToHex(b.PrevBlock.ClientStateHash)))
			return err
		}

		// Reset block state for retry.
		b.Txns = nil
		b.ClientState = nil
		return mc.generateBlock(ctx, b, minerChain, waitOver, waitC)
	})
}

func (mc *Chain) createGenerateChallengeTxn(b *block.Block) (*transaction.Transaction, error) {
	return mc.createGenChalTxn(b)
}
