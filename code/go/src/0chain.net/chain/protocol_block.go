package chain

import (
	"context"

	"0chain.net/block"
	"0chain.net/common"
	. "0chain.net/logging"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*ComputeFinalizedBlock - compute the block that has been finalized. It should be the one in the prior round
TODO: This logic needs refinement when the sharders start saving only partial set of blocks they are responsible for
*/
func (c *Chain) ComputeFinalizedBlock(ctx context.Context, r *round.Round) *block.Block {
	tips := r.GetNotarizedBlocks()
	if len(tips) == 0 {
		// This can happen due to network or joining in the middle scenario
		return nil
	}
	for true {
		ntips := make([]*block.Block, 0, 1)
		for _, b := range tips {
			if b.PrevBlock == nil {
				Logger.Debug("compute finalized block: null prev block", zap.Any("round", r.Number), zap.Any("block_round", b.Round), zap.Any("block", b.Hash))
				return nil
			}
			found := false
			for _, nb := range ntips {
				if b.PrevHash == nb.Hash {
					found = true
					break
				}
			}
			if found {
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
	if fb.Round == r.Number {
		return nil
	}
	return fb
}

/*VerifyTicket - verify the ticket */
func (c *Chain) VerifyTicket(ctx context.Context, b *block.Block, bvt *block.VerificationTicket) error {
	sender := c.Miners.GetNode(bvt.VerifierID)
	if sender == nil {
		return common.InvalidRequest("Verifier unknown or not authorized at this time")
	}

	if ok, _ := sender.Verify(bvt.Signature, b.Hash); !ok {
		return common.InvalidRequest("Couldn't verify the signature")
	}
	return nil
}

/*VerifyNotarization - verify that the notarization is correct */
func (c *Chain) VerifyNotarization(ctx context.Context, b *block.Block, bvt []*block.VerificationTicket) error {
	if b.Round != 0 && bvt == nil {
		return common.NewError("no_verification_tickets", "No verification tickets for this block")
	}
	// TODO: Logic similar to IsBlockNotarized to check the count satisfies (refactor)

	signMap := make(map[string]bool, len(bvt))
	for _, vt := range bvt {
		sign := vt.Signature
		_, signExists := signMap[sign]
		if signExists {
			return common.NewError("duplicate_ticket_signature", "Found duplicate signature for verification ticket of the block")
		}
		signMap[sign] = true
	}

	for _, vt := range bvt {
		if err := c.VerifyTicket(ctx, b, vt); err != nil {
			return err
		}
	}
	return nil
}

/*IsBlockNotarized - Does the given number of signatures means eligible for notraization?
TODO: For now, we just assume more than 50% */
func (c *Chain) IsBlockNotarized(ctx context.Context, b *block.Block) bool {
	numSignatures := b.GetVerificationTicketsCount()
	if 3*numSignatures >= 2*c.Miners.Size() {
		return true
	}
	return false
}
