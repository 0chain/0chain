package chain

import (
	"context"

	"0chain.net/node"

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
	if b.Round == 0 {
		return nil
	} else if bvt == nil {
		return common.NewError("no_verification_tickets", "No verification tickets for this block")
	}

	numSignatures := len(bvt)
	if numSignatures < c.GetNotarizationThresholdCount() {
		return common.NewError("block_not_notarized", "Number of Verification tickets for the block are less than notarization threshold count")
	}

	signMap := make(map[string]bool, numSignatures)
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

/*IsBlockNotarized - Does the given number of signatures means eligible for notarization? */
func (c *Chain) IsBlockNotarized(ctx context.Context, b *block.Block) bool {
	numSignatures := b.GetVerificationTicketsCount()
	if numSignatures >= c.GetNotarizationThresholdCount() {
		return true
	}
	return false
}

/*UpdateNodeState - based on the incoming valid blocks, update the nodes that notarized the block to be active
 Useful to increase the speed of node status discovery which increases the reliablity of the network
Simple 3 miner scenario :

1) a discovered b & c.
2) b discovered a.
3) b and c are yet to discover each other
4) a generated a block and sent it to b & c, got it notarized send next round started
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
