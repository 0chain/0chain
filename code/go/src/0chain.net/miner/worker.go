package miner

import (
	"context"
	"fmt"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*SetupWorkers - Setup the miner's workers */
func SetupWorkers() {
	ctx := common.GetRootContext()
	go GetMinerChain().BlockWorker(ctx)
}

/*BlockWorker - a job that does all the work related to blocks in each round */
func (mc *Chain) BlockWorker(ctx context.Context) {
	for true {
		select {
		case <-ctx.Done():
			return
		case msg := <-mc.GetBlockMessageChannel():
			fmt.Printf("received message: %v\n", msg.Type)
			switch msg.Type {
			case MessageVerify:
				mc.HandleVerifyBlockMessage(ctx, msg)
			case MessageVerificationTicket:
				mc.HandleVerificationTicketMessage(ctx, msg)
			case MessageConsensus:
				mc.HandleConsensusMessage(ctx, msg)
			}
		}
	}
}

func (mc *Chain) HandleVerifyBlockMessage(ctx context.Context, msg *BlockMessage) {
	bvt, err := mc.VerifyBlock(ctx, msg.Block)
	if err != nil {
		return
	}
	mc.SendVerificationTicket(ctx, msg.Block, bvt)
}

func (mc *Chain) HandleVerificationTicketMessage(ctx context.Context, msg *BlockMessage) {
	err := mc.VerifyTicket(ctx, msg.Block, msg.BlockVerificationTicket)
	if err != nil {
		return
	}
	if mc.AddVerificationTicket(ctx, msg.Block, &msg.BlockVerificationTicket.VerificationTicket) {
		if mc.ReachedConsensus(ctx, msg.Block) {
			consensus := datastore.GetEntityMetadata("block_consensus").Instance().(*Consensus)
			consensus.BlockID = msg.Block.Hash
			consensus.VerificationTickets = msg.Block.VerificationTickets
			mc.SendConsensus(ctx, consensus)
			//TODO: Finalize previous round
		}
	}
}

func (mc *Chain) HandleConsensusMessage(ctx context.Context, msg *BlockMessage) {
	//TODO: Finalize previous round
}
