package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/datastore"
)

/*SendBlock - send the block proposal to the network */
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	m2m.SendAll(VerifyBlockSender(b))
}

// TODO (sfxdx): TO REMOVE -- OLD CODE
//
//
/*
func (mc *Chain) kickJoiningMiners(ctx context.Context, mb *block.MagicBlock,
	b *block.Block) {

	// kick new miner on VC

	var (
		lfb = mc.GetLatestFinalizedBlock() //
		nvc = mc.NextViewChange(b)         // use this block ?
	)

	if b.Round < nvc || b.Round > nvc+chain.ViewChangeOffset {
		return
	}

	// after block with new magic block
	var nmb = mc.GetMagicBlock(b.Round + chain.ViewChangeOffset)
	if nmb.StartingRound == mb.StartingRound {
		return // not a new MB
	}

	// new miners set
	var (
		miners    = mb.Miners
		nminers   = nmb.Miners
		newminers []string

		from = lfb.Round + 1
	)

	// skipping miners already send
	for _, n := range nminers.CopyNodes() {
		var key = n.GetKey()
		if miners.HasNode(key) {
			continue // already send the notarization
		}
		newminers = append(newminers, key)
	}

	if len(newminers) == 0 {
		return
	}

	println("kickJoiningMiners", b.Round, len(newminers))

	// send all joining notarizations one by one
	for i := from; i <= b.Round; i++ {
		var r = mc.GetMinerRound(i)
		if r == nil {
			continue
		}
		var bx = r.GetHeaviestNotarizedBlock()
		for _, key := range newminers {
			go nminers.SendTo(MinerNotarizedBlockSender(bx), key)
		}
	}

	// the notarization
	for _, key := range newminers {
		go nminers.SendTo(MinerNotarizedBlockSender(b), key)
		// go nminers.SendTo(BlockNotarizationSender(not), key)
	}

}
*/

// SendNotarization - send the block notarization (collection of verification
// tickets enough to say notarization is reached).
func (mc *Chain) SendNotarization(ctx context.Context, b *block.Block) {
	var notarization = datastore.GetEntityMetadata("block_notarization").
		Instance().(*Notarization)

	notarization.BlockID = b.Hash
	notarization.Round = b.Round
	notarization.VerificationTickets = b.GetVerificationTickets()
	notarization.Block = b

	// magic block of current miners set
	var (
		mb     = mc.GetMagicBlock(b.Round)
		miners = mb.Miners
	)

	go miners.SendAll(BlockNotarizationSender(notarization))
	mc.SendNotarizedBlock(ctx, b)

	// TODO (sfxdx): DON'T PUSH -- PULL!
	//
	// kick new miners joining on VC
	// mc.kickJoiningMiners(ctx, mb, b)
}

// SendNotarizedBlock - send the notarized block.
func (mc *Chain) SendNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		mb := mc.GetMagicBlock(b.Round)
		m2s := mb.Sharders
		m2s.SendAll(NotarizedBlockSender(b))
	}
}

// ForcePushNotarizedBlock pushes notarized blocks to sharders.
func (mc *Chain) ForcePushNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		mb := mc.GetMagicBlock(b.Round)
		m2s := mb.Sharders
		m2s.SendAll(NotarizedBlockForcePushSender(b))
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

// TODO (sfxdx): TO REMOVE -- DEAD CODE
//
// /*SendNotarizedBlockToMiners - send a notarized block to a miner */
// func (mc *Chain) SendNotarizedBlockToMiners(ctx context.Context, b *block.Block) {
// 	mb := mc.GetMagicBlock(b.Round)
// 	m2m := mb.Miners
// 	m2m.SendAll(MinerNotarizedBlockSender(b))
// }

// TODO (sfxdx): to remove
//
//
// // SendNotarizedBlockToMiners - send a notarized block to a miner from pool
// func (mc *Chain) SendNotarizedBlockToPoolNodes(ctx context.Context, b *block.Block,
// 	pool *node.Pool, nodes []*node.Node, retry int) {
//
// 	if retry <= 0 {
// 		retry = 1
// 	}
// 	sendTo := nodes
// 	for retry > 0 {
// 		sentTo := pool.SendToMultipleNodes(MinerNotarizedBlockSender(b), sendTo)
// 		if len(sentTo) == len(nodes) {
// 			break
// 		}
// 		retry--
// 		if len(sentTo) > 0 {
// 			sentMap := make(map[string]struct{}, len(sentTo))
// 			for _, sentNode := range sentTo {
// 				sentMap[sentNode.ID] = struct{}{}
// 			}
// 			newSendNode := make([]*node.Node, 0, len(sendTo)-len(sentMap))
// 			for _, sendNode := range sentTo {
// 				if _, found := sentMap[sendNode.ID]; !found {
// 					newSendNode = append(newSendNode, sendNode)
// 				}
// 			}
// 			sendTo = newSendNode
// 		}
// 		time.Sleep(time.Second)
// 	}
// }
