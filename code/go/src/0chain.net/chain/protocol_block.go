package chain

import (
	"context"

	"0chain.net/datastore"
	"0chain.net/node"

	"0chain.net/block"
	"0chain.net/common"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*VerifyTicket - verify the ticket */
func (c *Chain) VerifyTicket(ctx context.Context, blockHash string, bvt *block.VerificationTicket) error {
	sender := c.Miners.GetNode(bvt.VerifierID)
	if sender == nil {
		return common.InvalidRequest("Verifier unknown or not authorized at this time")
	}

	if ok, _ := sender.Verify(bvt.Signature, blockHash); !ok {
		return common.InvalidRequest("Couldn't verify the signature")
	}
	return nil
}

/*VerifyNotarization - verify that the notarization is correct */
func (c *Chain) VerifyNotarization(ctx context.Context, blockHash string, bvt []*block.VerificationTicket) error {
	if bvt == nil {
		return common.NewError("no_verification_tickets", "No verification tickets for this block")
	}
	signMap := make(map[string]bool, len(bvt))
	for _, vt := range bvt {
		sign := vt.Signature
		_, signExists := signMap[sign]
		if signExists {
			return common.NewError("duplicate_ticket_signature", "Found duplicate signature for verification ticket of the block")
		}
		signMap[sign] = true
	}
	if !c.reachedNotarization(bvt) {
		return common.NewError("block_not_notarized", "Verification tickets not sufficient to reach notarization")
	}
	for _, vt := range bvt {
		if err := c.VerifyTicket(ctx, blockHash, vt); err != nil {
			return err
		}
	}
	return nil
}

/*IsBlockNotarized - Does the given number of signatures means eligible for notarization? */
func (c *Chain) IsBlockNotarized(ctx context.Context, b *block.Block) bool {
	return c.reachedNotarization(b.VerificationTickets)
}

func (c *Chain) reachedNotarization(bvt []*block.VerificationTicket) bool {
	if c.ThresholdByCount > 0 {
		numSignatures := len(bvt)
		if numSignatures < c.GetNotarizationThresholdCount() {
			return false
		}
	}
	if c.ThresholdByStake > 0 {
		verifiersStake := 0
		for _, ticket := range bvt {
			verifiersStake += c.getMiningStake(ticket.VerifierID)
		}
		if verifiersStake < c.ThresholdByStake {
			return false
		}
	}
	return true
}

/*ComputeFinalizedBlock - compute the block that has been finalized */
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, r round.RoundI) *block.Block {
	isIn := func(blocks []*block.Block, hash string) bool {
		for _, b := range blocks {
			if b.Hash == hash {
				return true
			}
		}
		return false
	}
	roundNumber := r.GetRoundNumber()
	tips := r.GetNotarizedBlocks()
	if len(tips) == 0 {
		Logger.Error("compute finalize block: no notarized blocks", zap.Int64("round", r.GetRoundNumber()))
		return nil
	}
	for true {
		ntips := make([]*block.Block, 0, 1)
		for _, b := range tips {
			if b.PrevBlock == nil {
				pb := c.GetPreviousBlock(ctx, b)
				if pb == nil {
					Logger.Error("compute finalized block: null prev block", zap.Int64("round", roundNumber), zap.Int64("block_round", b.Round), zap.String("block", b.Hash))
					return nil
				}
			}
			if isIn(ntips, b.PrevHash) {
				continue
			}
			ntips = append(ntips, b.PrevBlock)
		}
		tips = ntips
		if len(tips) == 1 {
			break
		}
	}
	if len(tips) != 1 {
		return nil
	}
	fb := tips[0]
	if fb.Round == r.GetRoundNumber() {
		return nil
	}
	return fb
}

/*UpdateNodeState - based on the incoming valid blocks, update the nodes that notarized the block to be active
 Useful to increase the speed of node status discovery which increases the reliablity of the network
Simple 3 miner scenario :

1) a discovered b & c.
2) b discovered a.
3) b and c are yet to discover each other
4) a generated a block and sent it to b & c, got it notarized and next round started
5) c is the generator who generated the block. He will only send it to a as b is not discovered to be active.
    But if the prior block has b's signature (may or may not, but if it did), c can discover b is active before generating the block and so will send it to b
*/
func (c *Chain) UpdateNodeState(b *block.Block) {
	for _, vt := range b.VerificationTickets {
		signer := c.Miners.GetNode(vt.VerifierID)
		if signer == nil {
			Logger.Error("this should not happen!")
			continue
		}
		if signer.Status != node.NodeStatusActive {
			signer.Status = node.NodeStatusActive
		}
	}
}

/*GetNotarizedBlock - get a notarized block for a round */
func (c *Chain) GetNotarizedBlock(blockHash string) *block.Block {
	nbrequestor := MinerNotarizedBlockRequestor
	cround := c.CurrentRound
	params := map[string]string{"block": blockHash}
	ctx := common.GetRootContext()
	var b *block.Block
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Info("get notarized block", zap.String("block", blockHash), zap.Int64("cround", cround), zap.Int64("current_round", c.CurrentRound))
		nb, ok := entity.(*block.Block)
		if !ok {
			return nil, common.NewError("invalid_entity", "Invalid entity")
		}
		if err := c.VerifyNotarization(ctx, nb.Hash, nb.VerificationTickets); err != nil {
			Logger.Error("get notarized block - validate notarization", zap.String("block", blockHash), zap.Error(err))
			return nil, err
		}
		if err := nb.Validate(ctx); err != nil {
			Logger.Error("get notarized block - validate", zap.String("block", blockHash), zap.Any("block_obj", nb), zap.Error(err))
			return nil, err
		}
		b = c.AddBlock(nb)
		r := c.GetRound(b.Round)
		if r == nil {
			Logger.Error("get notarized block - no round (TODO)", zap.String("block", blockHash), zap.Int64("round", b.Round), zap.Int64("cround", cround), zap.Int64("current_round", c.CurrentRound))
		}
		if r != nil {
			b, _ = r.AddNotarizedBlock(b)
		}
		Logger.Info("get notarized block", zap.Int64("round", b.Round), zap.String("block", b.Hash))
		return b, nil
	}
	n2n := c.Miners
	n2n.RequestEntity(ctx, nbrequestor, params, handler)
	return b
}

/*GetPreviousBlock - get the previous block from the network */
func (c *Chain) GetPreviousBlock(ctx context.Context, b *block.Block) *block.Block {
	if b.PrevBlock != nil {
		return b.PrevBlock
	}
	pb, err := c.GetBlock(ctx, b.PrevHash)
	if err == nil {
		b.SetPreviousBlock(pb)
		return pb
	}
	blocks := make([]*block.Block, 0, 10)
	Logger.Info("fetch previous block", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
	cb := b
	for idx := 0; idx < 10; idx++ {
		Logger.Info("fetching previous block", zap.Int("idx", idx), zap.Int64("cround", cb.Round), zap.String("cblock", cb.Hash), zap.String("cprev_block", cb.PrevHash))
		nb := c.GetNotarizedBlock(cb.PrevHash)
		if nb == nil {
			Logger.Error("get previous block (unable to get prior blocks)", zap.Int("idx", idx), zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("cround", cb.Round), zap.String("cblock", cb.Hash), zap.String("cprev_block", cb.PrevHash))
			return nil
		}
		cb = nb
		blocks = append(blocks, cb)
		pb, err = c.GetBlock(ctx, cb.PrevHash)
		if pb != nil {
			cb.SetPreviousBlock(pb)
			break
		}
	}
	if cb.PrevBlock == nil { // This happens after fetching as far as per the previous for loop and still not having the prior block
		Logger.Error("get previous block (missing continuity)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("oldest_fetched_round", cb.Round), zap.String("oldest_fetched_block", cb.Hash), zap.String("missing_prior_block", cb.PrevHash))
		return nil
	}
	for idx := len(blocks) - 1; idx >= 0; idx-- {
		cb := blocks[idx]
		if cb.PrevBlock == nil {
			pb, err := c.GetBlock(ctx, cb.PrevHash)
			if err != nil {
				Logger.Error("get previous block (missing continuity)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int64("cb_round", cb.Round), zap.String("cb_block", cb.Hash), zap.String("missing_prior_block", cb.PrevHash))
				return nil
			}
			cb.SetPreviousBlock(pb)
		}
		c.ComputeState(ctx, cb)
	}
	pb, err = c.GetBlock(ctx, b.PrevHash)
	if err == nil {
		b.SetPreviousBlock(pb)
	}
	return pb
}

//Note: this is expected to work only for small forks
func (c *Chain) commonAncestor(ctx context.Context, b1 *block.Block, b2 *block.Block) *block.Block {
	if b1 == nil || b2 == nil {
		return nil
	}
	if b1 == b2 || b1.Hash == b2.Hash {
		return b1
	}
	if b2.Round < b1.Round {
		b1, b2 = b2, b1
	}
	for b2.Round != b1.Round {
		b2 = c.GetPreviousBlock(ctx, b2)
		if b2 == nil {
			return nil
		}
	}
	for b1 != b2 {
		b1 = c.GetPreviousBlock(ctx, b1)
		if b1 == nil {
			return nil
		}
		b2 = c.GetPreviousBlock(ctx, b2)
		if b2 == nil {
			return nil
		}
	}
	return b1
}
