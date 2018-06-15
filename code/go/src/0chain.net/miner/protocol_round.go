package miner

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/memorystore"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
)

const DELTA = 200 * time.Millisecond
const BLOCK_TIME = 3 * DELTA
const FINALIZATION_TIME = 2 * DELTA

/*GetBlockToExtend - Get the block to extend from the given round */
func (mc *Chain) GetBlockToExtend(r *Round) *block.Block {
	rnb := r.GetNotarizedBlocks()
	if len(rnb) > 0 {
		if len(rnb) == 1 {
			return rnb[0]
		}
		//TODO: pick the best possible block
		return rnb[0]
	}
	return nil
}

/*GenerateRoundBlock - given a round number generates a block*/
func (mc *Chain) GenerateRoundBlock(ctx context.Context, r *Round) (*block.Block, error) {
	pround := mc.GetRound(r.Number - 1)
	if pround == nil {
		Logger.Error("generate block (prior round not found)", zap.Any("round", r.Number-1))
		return nil, common.NewError("invalid_round,", "Round not available")
	}
	pb := mc.GetBlockToExtend(pround)
	if pb == nil {
		Logger.Error("generate block (prior block not found)", zap.Any("round", r.Number))
		return nil, common.NewError("block_gen_no_block_to_extend", "Do not have the block to extend this round")
	}
	b := datastore.GetEntityMetadata("block").Instance().(*block.Block)
	b.ChainID = mc.ID
	b.MagicBlockHash = mc.CurrentMagicBlock.Hash
	b.SetPreviousBlock(pb)
	err := mc.GenerateBlock(ctx, b)
	if err != nil {
		Logger.Error("generate block error", zap.Error(err))
		return nil, err
	}
	mc.AddToRoundVerification(ctx, r, b)
	mc.AddBlock(b)
	mc.SendBlock(ctx, b)
	return b, nil
}

/*AddToRoundVerification - Add a block to verify : WARNING: does not support concurrent access for a given round */
func (mc *Chain) AddToRoundVerification(ctx context.Context, mr *Round, b *block.Block) {
	if !mc.ValidateMagicBlock(ctx, b) {
		Logger.Info("invalid magic block", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Any("magic_block", b.MagicBlockHash))
		return
	}
	mc.AddBlock(b)
	vctx := mr.StartVerificationBlockCollection(ctx)
	if vctx != nil {
		go mc.CollectBlocksForVerification(vctx, mr)
	}
	mr.AddBlockToVerify(b)
}

/*CollectBlocksForVerification - keep collecting the blocks till timeout and then start verifying */
func (mc *Chain) CollectBlocksForVerification(ctx context.Context, r *Round) {
	var blockTimeTimer = time.NewTimer(BLOCK_TIME)
	var sendVerification = false
	verifyAndSend := func(ctx context.Context, r *Round, b *block.Block) bool {
		bvt, err := mc.VerifyRoundBlock(ctx, r, b)
		if err != nil {
			if err == ErrRoundMismatch {
				Logger.Info("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Any("current_round", mc.CurrentRound))
			} else {
				Logger.Error("verify round block", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Error(err))
			}
			return false
		}
		r.Block = b
		if b.MinerID != node.Self.GetKey() {
			mc.SendVerificationTicket(ctx, b, bvt)
		}
		mc.ProcessVerifiedTicket(ctx, r, b, &bvt.VerificationTicket)
		return true
	}
	var blocks = make([]*block.Block, 0, 10)
	for true {
		select {
		case <-ctx.Done():
			return
		case <-blockTimeTimer.C:
			sendVerification = true
			// Sort the accumulated blocks by the rank and process them
			blocks = r.GetBlocksByRank(blocks)
			// Keep verifying all the blocks collected so far in the best rank order till the first
			// successul verification
			for _, b := range blocks {
				if verifyAndSend(ctx, r, b) {
					break
				}
			}
		case b := <-r.GetBlocksToVerifyChannel():
			if sendVerification {
				// Is this better than the current best block
				if r.Block == nil || b.RoundRank < r.Block.RoundRank {
					verifyAndSend(ctx, r, b)
				}
			} else { // Accumulate all the blocks into this array till the BlockTime timeout
				blocks = append(blocks, b)
			}
		}
	}
}

/*VerifyRoundBlock - given a block is verified for a round*/
func (mc *Chain) VerifyRoundBlock(ctx context.Context, r *Round, b *block.Block) (*block.BlockVerificationTicket, error) {
	if mc.CurrentRound != r.Number {
		return nil, ErrRoundMismatch
	}
	if b.MinerID == node.Self.GetKey() {
		return mc.SignBlock(ctx, b)
	}
	prevBlock, err := mc.GetBlock(ctx, b.PrevHash)
	if err != nil {
		//TODO: create previous round AND request previous block from miner who sent current block for verification
		Logger.Error("verify round", zap.Any("round", r.Number), zap.Any("block", b.Hash), zap.Any("prev_block", b.PrevHash), zap.Error(err))
		return nil, common.NewError("prev_block_error", "Error getting the previous block")
	}

	if prevBlock == nil {
		//TODO: create previous round AND request previous block from miner who sent current block for verification
		return nil, common.NewError("invalid_block", fmt.Sprintf("Previous block doesn't exist: %v", b.PrevHash))
	}

	/* Note: We are verifying the notrization of the previous block we have with
	   the prev verification tickets of the current block. This is right as all the
	   necessary verification tickets & notarization message may not have arrived to us */
	if err := mc.VerifyNotarization(ctx, prevBlock, b.PrevBlockVerficationTickets); err != nil {
		return nil, err
	}

	bvt, err := mc.VerifyBlock(ctx, b)
	if err != nil {
		return nil, err
	}
	return bvt, nil
}

/*ProcessVerifiedTicket - once a verified ticket is receiveid, do further processing with it */
func (mc *Chain) ProcessVerifiedTicket(ctx context.Context, r *Round, b *block.Block, vt *block.VerificationTicket) {
	if mc.AddVerificationTicket(ctx, b, vt) {
		if mc.IsBlockNotarized(ctx, b) {
			r.Block = b
			mc.CancelRoundVerification(ctx, r)
			notarization := datastore.GetEntityMetadata("block_notarization").Instance().(*Notarization)
			notarization.BlockID = b.Hash
			notarization.Round = b.Round
			notarization.VerificationTickets = b.VerificationTickets
			r.AddNotarizedBlock(b)
			mc.SendNotarization(ctx, notarization)
			if mc.GetRound(r.Number+1) == nil {
				nr := datastore.GetEntityMetadata("round").Instance().(*round.Round)
				nr.Number = r.Number + 1
				nr.RandomSeed = rand.New(rand.NewSource(r.RandomSeed)).Int63()
				nmr := mc.CreateRound(nr)
				// Even if the context is cancelled, we want to proceed with the next round, hence start with a root context
				go mc.startNewRound(common.GetRootContext(), nmr)
				mc.Miners.SendAll(RoundStartSender(nr))
			}
			pr := mc.GetRound(r.Number - 1)
			if pr != nil && pr.Block != nil {
				mc.FinalizeRound(ctx, pr)
			}
		}
	}
}

/*CancelRoundVerification - cancel verifications happening within a round */
func (mc *Chain) CancelRoundVerification(ctx context.Context, r *Round) {
	r.CancelVerification() // No need for further verification of any blocks
}

/*FinalizeRound - starting from the given round work backwards and identify the round that can be
  assumed to be finalized as only one chain has survived.
  Note: It is that round and prior that actually get finalized.
*/
func (mc *Chain) FinalizeRound(ctx context.Context, r *Round) {
	/*TODO: This is incorrect because when we ask r to finalize, it's actually the r-1 round's block that gets finalized */
	if r.IsFinalized() {
		return
	}
	var finzalizeTimer = time.NewTimer(FINALIZATION_TIME)
	select {
	case <-finzalizeTimer.C:
		break
	}
	fb := mc.ComputeFinalizedBlock(ctx, &r.Round)
	if fb == nil {
		Logger.Info("finalization - no decisive block to finalize yet", zap.Any("round", r.Number))
		return
	}
	lfbHash := mc.LatestFinalizedBlock.Hash
	mc.LatestFinalizedBlock = fb
	frchain := make([]*block.Block, 0, 1)
	for b := fb; b != nil && b.Hash != lfbHash; b = b.PrevBlock {
		frchain = append(frchain, b)
	}
	deadBlocks := make([]*block.Block, 0, 1)
	for idx := range frchain {
		fb = frchain[len(frchain)-1-idx]
		fr := mc.GetRound(fb.Round)
		Logger.Info("finalizing round", zap.Any("round", r.Number), zap.Any("finalized_round", fb.Round), zap.Any("hash", fb.Hash))
		fr.Finalize(fb)
		mc.UpdateFinalizedBlock(ctx, fb)
		mc.SendFinalizedBlock(ctx, fb)
		frb := mc.GetRoundBlocks(r.Number)
		for _, b := range frb {
			if b.Hash != fb.Hash {
				deadBlocks = append(deadBlocks, b)
			}
		}
	}
	// Prune all the dead blocks
	go func() {
		for _, b := range deadBlocks {
			mc.DeleteBlock(ctx, b)
		}
	}()
}

/*UpdateFinalizedBlock - update the latest finalized block */
func (mc *Chain) UpdateFinalizedBlock(ctx context.Context, b *block.Block) {
	if b.Hash == mc.LatestFinalizedBlock.Hash {
		return
	}
	txnEntityMetadata := datastore.GetEntityMetadata("txn")
	ctx = memorystore.WithEntityConnection(ctx, txnEntityMetadata)
	defer memorystore.Close(ctx)
	mc.FinalizeBlock(ctx, b)
}
