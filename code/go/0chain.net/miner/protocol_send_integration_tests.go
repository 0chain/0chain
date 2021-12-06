//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"log"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
	"0chain.net/core/common"
)

func getBadVRFS(vrfs *round.VRFShare) (bad *round.VRFShare) {
	bad = new(round.VRFShare)
	*bad = *vrfs
	bad.Share = revertString(bad.Share) // bad share
	return
}

func withTimeout(vrfs *round.VRFShare, timeout int) (bad *round.VRFShare) {
	bad = new(round.VRFShare)
	*bad = *vrfs
	bad.RoundTimeoutCount = timeout
	return
}

func (mc *Chain) SendVRFShare(ctx context.Context, vrfs *round.VRFShare) {
	var (
		mb        = mc.GetMagicBlock(vrfs.Round)
		state     = crpc.Client().State()
		badVRFS   *round.VRFShare
		good, bad []*node.Node
	)

	// not possible to send bad VRFS and bad round timeout at the same time
	switch {
	case state.VRFS != nil:
		badVRFS = getBadVRFS(vrfs)
		good, bad = crpcutils.Split(state, state.VRFS, mb.Miners.CopyNodes())
	case state.RoundTimeout != nil:
		badVRFS = withTimeout(vrfs, vrfs.RoundTimeoutCount+1) // just increase
		good, bad = crpcutils.Split(state, state.RoundTimeout,
			mb.Miners.CopyNodes())
	default:
		good = mb.Miners.CopyNodes() // all good
	}

	if len(good) > 0 {
		mb.Miners.SendToMultipleNodes(ctx, RoundVRFSender(vrfs), good)
	}
	if len(bad) > 0 {
		mb.Miners.SendToMultipleNodes(ctx, RoundVRFSender(badVRFS), bad)
	}
}

func getBadBVTHash(ctx context.Context, b *block.Block) (
	bad *block.BlockVerificationTicket) {

	bad = new(block.BlockVerificationTicket)
	bad.BlockID = b.Hash
	bad.Round = b.Round
	var (
		self = node.Self
		err  error
	)
	bad.VerifierID = self.Underlying().GetKey()
	bad.Signature, err = self.Sign(revertString(b.Hash)) // wrong hash
	if err != nil {
		panic(err)
	}
	return
}

func getBadBVTKey(ctx context.Context, b *block.Block) (
	bad *block.BlockVerificationTicket) {

	bad = new(block.BlockVerificationTicket)
	bad.BlockID = b.Hash
	bad.Round = b.Round
	var (
		selfNodeKey = node.Self.Underlying().GetKey()
		err         error
	)
	bad.VerifierID = selfNodeKey
	bad.Signature, err = crpcutils.Sign(b.Hash) // wrong private key
	if err != nil {
		panic(err)
	}
	return
}

// SendVerificationTicket - send the block verification ticket
func (mc *Chain) SendVerificationTicket(ctx context.Context, b *block.Block,
	bvt *block.BlockVerificationTicket) {

	var (
		mb          = mc.GetMagicBlock(b.Round)
		state       = crpc.Client().State()
		selfNodeKey = node.Self.Underlying().GetKey()

		good, bad []*node.Node
	)

	if mc.VerificationTicketsTo == chain.Generator && b.MinerID != selfNodeKey {
		switch {
		case state.WrongVerificationTicketHash != nil:
			// (wrong hash)
			if state.WrongVerificationTicketHash.IsGood(state, b.MinerID) {
				mb.Miners.SendTo(ctx, VerificationTicketSender(bvt), b.MinerID)
			} else if state.WrongVerificationTicketHash.IsBad(state, b.MinerID) {
				var badvt = getBadBVTHash(ctx, b)
				mb.Miners.SendTo(ctx, VerificationTicketSender(badvt), b.MinerID)
			}
		case state.WrongVerificationTicketKey != nil:
			// (wrong secret key)
			if state.WrongVerificationTicketKey.IsGood(state, b.MinerID) {
				mb.Miners.SendTo(ctx, VerificationTicketSender(bvt), b.MinerID)
			} else if state.WrongVerificationTicketKey.IsBad(state, b.MinerID) {
				var badvt = getBadBVTKey(ctx, b)
				mb.Miners.SendTo(ctx, VerificationTicketSender(badvt), b.MinerID)
			}
		default:
			// (usual sending)
			mb.Miners.SendTo(ctx, VerificationTicketSender(bvt), b.MinerID)
		}
		return
	}

	var badvt *block.BlockVerificationTicket

	switch {
	case state.WrongVerificationTicketHash != nil:
		// (wrong hash)
		badvt = getBadBVTHash(ctx, b)
		good, bad = crpcutils.Split(state, state.WrongVerificationTicketHash,
			mb.Miners.CopyNodes())
	case state.WrongVerificationTicketKey != nil:
		// (wrong secret key)
		badvt = getBadBVTKey(ctx, b)
		good, bad = crpcutils.Split(state, state.WrongVerificationTicketKey,
			mb.Miners.CopyNodes())
	default:
	}

	if badvt == nil {
		mb.Miners.SendAll(ctx, VerificationTicketSender(bvt)) // (usual sending)
		return
	}

	if len(good) > 0 {
		mb.Miners.SendToMultipleNodes(ctx, VerificationTicketSender(bvt), good)
	}
	if len(bad) > 0 {
		mb.Miners.SendToMultipleNodes(ctx, VerificationTicketSender(badvt), bad)
	}
}

// SendBlock - send the block proposal to the network.
func (mc *Chain) SendBlock(ctx context.Context, b *block.Block) {
	if mc.isSendingDifferentBlocks(b.Round) {
		mc.sendDifferentBlocksToMiners(ctx, b)
		return
	}

	mc.sendBlock(ctx, b)
}

func (mc *Chain) sendDifferentBlocksToMiners(ctx context.Context, b *block.Block) {
	miners := mc.GetMagicBlock(b.Round).Miners.CopyNodes()
	blocks, err := randomizeBlocks(b, len(miners))
	if err != nil {
		log.Panicf("Conductor: SendDifferentBlocksToMiners: error while randomizing blocks: %v", err)
	}
	for ind, n := range miners {
		if n.ID == node.Self.ID {
			continue
		}

		b := blocks[ind]
		handler := VerifyBlockSender(b)
		ok := handler(ctx, n)
		if !ok {
			log.Panicf("Conductor: SendDifferentBlocksToMiners: block is not sent to miner with ID %s.", n.ID)
		}
	}

	err = configureSendDifferentBlocksToMiners(
		&cases.SendDifferentBlocksToMinersConfig{
			MinersRoundRank: mc.GetRound(b.Round).GetMinerRank(node.Self.Node),
		},
	)
	if err != nil {
		log.Panicf("Conductor: SendDifferentBlocksToMiners: error while configuring test case: " + err.Error())
	}
}

func configureSendDifferentBlocksToMiners(cfg *cases.SendDifferentBlocksToMinersConfig) error {
	caseCfg, err := cfg.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().ConfigureTestCase(caseCfg)
}

func (mc *Chain) isSendingDifferentBlocks(r int64) bool {
	isFirstGenerator := mc.GetRound(r).GetMinerRank(node.Self.Node) == 0
	testCfg := crpc.Client().State().SendDifferentBlocksToMiners
	return testCfg != nil && testCfg.Round == r && isFirstGenerator
}

func randomizeBlocks(b *block.Block, numBlocks int) ([]*block.Block, error) {
	blocks := make([]*block.Block, numBlocks)
	for ind := range blocks {
		cpBl, err := copyBlock(b)
		if err != nil {
			return nil, err
		}
		cpBl.CreationDate += common.Timestamp(ind)
		cpBl.HashBlock()
		blocks[ind] = cpBl
	}
	return blocks, nil
}

func copyBlock(b *block.Block) (*block.Block, error) {
	cp := new(block.Block)
	if err := cp.Decode(b.Encode()); err != nil {
		return nil, err
	}
	return cp, nil
}
