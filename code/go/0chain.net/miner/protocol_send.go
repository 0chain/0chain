package miner

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/core/datastore"
)

// SendBlock - send the block proposal to the network.
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	mb := mc.GetMagicBlock(b.Round)
	m2m := mb.Miners
	m2m.SendAll(VerifyBlockSender(b))
}

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
}

func newSharders(mb, nmb *block.MagicBlock) (ns []*node.Node) {
	ns = make([]*node.Node, 0, nmb.Sharders.Size())
	for key, node := range nmb.Sharders.NodesMap {
		if _, ok := mb.Sharders.NodesMap[key]; ok {
			continue
		}
		ns = append(ns, node)
	}
	return
}

// The newShardersMagicBlock returns magic block that is not view changed yet
// but going to be view changed soon. It useful at view change rounds and
// useless for any other case. If it returns nil, the we just does nothing.
// If it returns a magic block, the we should extract new sharders from it
// (that not joined yet) and push the block to them. Since, the block is not
// finalized yet, we (1) can't use GetMagicBlock to get the new magic block
// (2) should push the block to new sharders (to avoid corresponding pulling
// their side). If we skip this steps, then new sharders pull the magic block
// from miners (e.g. nothing critical). This method used to bootstrap them (the
// new, joining, sharders) on view change.
func (mc *Chain) newShardersMagicBlock(b *block.Block) (nmb *block.MagicBlock) {

	var (
		vco int64 = chain.ViewChangeOffset // short hand
		nvc       = mc.NextViewChange()    //
		rn        = b.Round                // short hand
	)

	// before next view change, or after it
	if rn < nvc || rn > nvc+vco {
		return // shouldn't
	}

	// so, magic block expected at nvc
	for b != nil && b.Round > nvc {
		b = b.PrevBlock // just get previous block
	}

	if b == nil || b.MagicBlock == nil {
		return // no block or no magic block (we can't continue)
	}

	return b.MagicBlock // the new, magic block
}

// The sendToNewSharders sends the given notarized block to new sharders
// at view change. The mb is current magic block.
func (mc *Chain) sendToNewSharders(mb *block.MagicBlock, b *block.Block) {
	var nmb = mc.newShardersMagicBlock(b)
	if nmb == nil {
		return // nothing to bootstrap, not a view changing
	}
	var ns = newSharders(mb, nmb)
	if len(ns) > 0 {
		nmb.Sharders.SendToMultiple(NotarizedBlockSender(b), ns)
	}
	// TODO (sfxdx): REMOVE THE INSPECTION
	{
		if len(ns) > 0 {
			println("NEW SHARDERS AT", b.Round)
			for _, node := range ns {
				println("  -", node.GetN2NURLBase())
			}
		} else {
			println("NO NEW SHARDERS AT", b.Round)
		}
	}
}

// SendNotarizedBlock - send the notarized block.
func (mc *Chain) SendNotarizedBlock(ctx context.Context, b *block.Block) {
	if mc.BlocksToSharder == chain.NOTARIZED {
		var (
			mb  = mc.GetMagicBlock(b.Round)
			mbs = mb.Sharders
		)
		mbs.SendAll(NotarizedBlockSender(b))
		// send to sharders from new magic block (if any) at view change rounds
		// (e.g. at 501-504); e.g. bootstrap new sharders at view change
		mc.sendToNewSharders(mb, b)
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
