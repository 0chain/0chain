package miner

import (
	"context"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
)

/*SendVRFShare - send the round vrf share */
func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	mb := mc.GetMagicBlock(vrfs.Round)
	m2m := mb.Miners
	m2m.SendAll(RoundVRFSender(vrfs))
}

/*SendBlock - send the block proposal to the network */
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	m2m.SendAll(VerifyBlockSender(b))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	if mc.VerificationTicketsTo == chain.Generator {
		if b.MinerID != node.Self.Underlying().GetKey() {
			m2m.SendTo(VerificationTicketSender(bvt), b.MinerID)
		}
	} else {
		m2m.SendAll(VerificationTicketSender(bvt))
	}
}

/*SendNotarization - send the block notarization (collection of verification tickets enough to say notarization is reached) */
func (mc *Chain) SendNotarization(ctx context.Context, b *block.Block) {
	notarization := datastore.GetEntityMetadata("block_notarization").Instance().(*Notarization)
	notarization.BlockID = b.Hash
	notarization.Round = b.Round
	notarization.VerificationTickets = b.GetVerificationTickets()
	notarization.Block = b
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	go m2m.SendAll(BlockNotarizationSender(notarization))
	mc.SendNotarizedBlock(ctx, b)
}

/*SendNotarizedBlock - send the notarized block */
func (mc *Chain) SendNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		mb := mc.GetMagicBlock(b.Round)
		m2s := mb.Sharders
		m2s.SendAll(NotarizedBlockSender(b))
	}
}

/*SendFinalizedBlock - send the finalized block to the sharders */
func (mc *Chain) SendFinalizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.FINALIZED {
		mb := mc.GetMagicBlock(b.Round)
		m2s := mb.Sharders
		m2s.SendAll(FinalizedBlockSender(b))
	}
}

/*SendNotarizedBlockToMiners - send a notarized block to a miner */
func (mc *Chain) SendNotarizedBlockToMiners(ctx context.Context, b *block.Block) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	m2m.SendAll(MinerNotarizedBlockSender(b))
}

/*SendNotarizedBlockToMiners - send a notarized block to a miner from pool */
func (mc *Chain) SendNotarizedBlockToPoolNodes(ctx context.Context, b *block.Block,
	pool *node.Pool, nodes []*node.Node, retry int) {
	if retry <= 0 {
		retry = 1
	}
	sendTo := nodes
	for retry > 0 {
		sentTo := pool.SendToMultipleNodes(MinerNotarizedBlockSender(b), sendTo)
		if len(sentTo) == len(nodes) {
			break
		}
		retry--
		if len(sentTo) > 0 {
			sentMap := make(map[string]struct{}, len(sentTo))
			for _, sentNode := range sentTo {
				sentMap[sentNode.ID] = struct{}{}
			}
			newSendNode := make([]*node.Node, 0, len(sendTo)-len(sentMap))
			for _, sendNode := range sentTo {
				if _, found := sentMap[sendNode.ID]; !found {
					newSendNode = append(newSendNode, sendNode)
				}
			}
			sendTo = newSendNode
		}
		time.Sleep(time.Second)
	}
}
