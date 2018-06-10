package miner

import (
	"context"

	"0chain.net/block"
	. "0chain.net/logging"
	"0chain.net/node"
	"0chain.net/round"
	"go.uber.org/zap"
)

/*SendRoundStart - send a new round start message */
func (mc *Chain) SendRoundStart(ctx context.Context, r *round.Round) {
	Logger.Info("Sending round start message", zap.Any("Node index", node.Self.Node.SetIndex), zap.Any("Key", node.Self.Node.GetKey()))
	//fmt.Printf("sending round start message from %v(%v)\n", node.Self.Node.SetIndex, node.Self.Node.GetKey())
	mc.Miners.SendAll(RoundStartSender(r))
}

/*SendBlock - send the generated block to the network */
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	Logger.Info("sending block proposal message from", zap.Any("Node index", node.Self.Node.SetIndex), zap.Any("Key", node.Self.Node.GetKey()))
	//fmt.Printf("sending block proposal message from %v(%v)\n", node.Self.Node.SetIndex, node.Self.Node.GetKey())
	mc.Miners.SendAll(VerifyBlockSender(b))
}

/*SendVerificationTicket - send the block verification ticket */
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block, bvt *block.BlockVerificationTicket) {
	Logger.Info("sending block verification ticket message from", zap.Any("Node index", node.Self.Node.SetIndex), zap.Any("Key", node.Self.Node.GetKey()))
	//fmt.Printf("sending block verification ticket message from %v(%v)\n", node.Self.Node.SetIndex, node.Self.Node.GetKey())
	mc.Miners.SendTo(VerificationTicketSender(bvt), b.MinerID)
}

/*SendNotarization - send the block notarization (collection of verification tickets enough to say notarization is reached) */
func (mc *Chain) SendNotarization(ctx context.Context, notarization *Notarization) {
	Logger.Info("sending block notarization message ", zap.Any("Node index", node.Self.Node.SetIndex), zap.Any("Key", node.Self.Node.GetKey()))
	//fmt.Printf("sending block notarization message from %v(%v)\n", node.Self.Node.SetIndex, node.Self.Node.GetKey())
	mc.Miners.SendAll(BlockNotarizationSender(notarization))
}

/*SendFinalizedBlock - send the finalized block to the sharders */
func (mc *Chain) SendFinalizedBlock(ctx context.Context, b *block.Block) {
	Logger.Info("sending finalized block message from", zap.Any("Node index", node.Self.Node.SetIndex), zap.Any("Key", node.Self.Node.GetKey()))
	//fmt.Printf("sending finalized block message from %v(%v)\n", node.Self.Node.SetIndex, node.Self.Node.GetKey())
	mc.Sharders.SendAll(FinalizedBlockSender(b))
}
