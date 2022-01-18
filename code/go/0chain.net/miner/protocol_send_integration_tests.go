//go:build integration_tests
// +build integration_tests

package miner

import (
	"context"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/conductor/cases"
	crpc "0chain.net/conductor/conductrpc"
	crpcutils "0chain.net/conductor/utils"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
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

func (mc *Chain) isTestingSendDifferentBlocks(round int64, timeoutCount int) bool {
	var (
		cfgFromFirstGen        = crpc.Client().State().SendDifferentBlocksFromFirstGenerator
		shouldTestFromFirstGen = cfgFromFirstGen != nil && cfgFromFirstGen.Round == round

		cfgFromAllGen        = crpc.Client().State().SendDifferentBlocksFromAllGenerators
		shouldTestFromAllGen = cfgFromAllGen != nil && cfgFromAllGen.Round == round
	)
	return (shouldTestFromAllGen || shouldTestFromFirstGen) && timeoutCount == 1
}

func addSendDifferentBlocksResult(roundTimeOutCount int) error {
	res := &cases.SendDiffBlocksResult{
		MinerID:      node.Self.ID,
		TimeoutCount: roundTimeOutCount,
	}
	blob, err := res.Encode()
	if err != nil {
		return err
	}
	return crpc.Client().AddTestCaseResult(blob)
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

	case mc.isTestingSendDifferentBlocks(vrfs.Round, vrfs.RoundTimeoutCount):
		log.Printf("adding tests result")
		if err := addSendDifferentBlocksResult(vrfs.RoundTimeoutCount); err != nil {
			log.Panicf("Conductor: error while adding test result: %v", err)
		}
		mc.sendVRFShare(ctx, vrfs)
		return

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

	if mc.VerificationTicketsTo() == chain.Generator && b.MinerID != selfNodeKey {
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
	if mc.isSendingDifferentBlocksFromFirstGenerator(b.Round) || mc.isSendingDifferentBlocksFromAllGenerators(b.Round) {
		mc.sendDifferentBlocks(ctx, b)
		return
	}

	mc.sendBlock(ctx, b)
}

func (mc *Chain) sendDifferentBlocks(ctx context.Context, b *block.Block) {
	miners := mc.GetMagicBlock(b.Round).Miners.CopyNodes()
	blocks, err := randomizeBlocks(b, len(miners))
	if err != nil {
		log.Panicf("Conductor: error while randomizing blocks: %v", err)
	}
	for ind, n := range miners {
		if n.ID == node.Self.ID {
			continue
		}

		b := blocks[ind]
		handler := VerifyBlockSender(b)
		ok := handler(ctx, n)
		if !ok {
			log.Panicf("Conductor: block is not sent to miner with ID %s", n.ID)
		}
	}
}

func (mc *Chain) isSendingDifferentBlocksFromFirstGenerator(r int64) bool {
	currRound := mc.GetRound(r)
	isFirstGenerator := currRound.GetMinerRank(node.Self.Node) == 0
	testCfg := crpc.Client().State().SendDifferentBlocksFromFirstGenerator
	return testCfg != nil && testCfg.Round == r && isFirstGenerator && currRound.GetTimeoutCount() == 0
}

func (mc *Chain) isSendingDifferentBlocksFromAllGenerators(r int64) bool {
	currRound := mc.GetRound(r)
	isGenerator := mc.IsRoundGenerator(mc.GetRound(r), node.Self.Node)
	testCfg := crpc.Client().State().SendDifferentBlocksFromAllGenerators
	return testCfg != nil && testCfg.Round == r && isGenerator && currRound.GetTimeoutCount() == 0
}

func randomizeBlocks(b *block.Block, numBlocks int) ([]*block.Block, error) {
	blocks := make([]*block.Block, numBlocks)
	for ind := range blocks {
		cpBl, err := randomizeBlock(b)
		if err != nil {
			return nil, err
		}
		blocks[ind] = cpBl
	}
	return blocks, nil
}

func randomizeBlock(b *block.Block) (*block.Block, error) {
	cpBl, err := copyBlock(b)
	if err != nil {
		return nil, err
	}

	txn, err := createDataTxn(encryption.Hash(strconv.Itoa(int(time.Now().UnixNano()))))
	if err != nil {
		return nil, err
	}
	cpBl.Txns = append(cpBl.Txns, txn)

	cpBl.HashBlock()
	if cpBl.Signature, err = node.Self.Sign(cpBl.Hash); err != nil {
		return nil, err
	}

	return cpBl, nil
}

func copyBlock(b *block.Block) (*block.Block, error) {
	blob, err := json.Marshal(b)
	if err != nil {
		return nil, err
	}
	cp := new(block.Block)
	if err := json.Unmarshal(blob, cp); err != nil {
		return nil, err
	}
	return cp, nil
}

func createDataTxn(data string) (*transaction.Transaction, error) {
	txn := &transaction.Transaction{
		VersionField: datastore.VersionField{
			Version: "1.0",
		},
		ClientID:        node.Self.ID,
		PublicKey:       node.Self.PublicKey,
		ChainID:         chain.GetServerChain().ID,
		TransactionData: data,
		CreationDate:    common.Now(),
		TransactionType: transaction.TxnTypeData,
	}
	txn.OutputHash = txn.ComputeOutputHash()
	if _, err := txn.Sign(node.Self.GetSignatureScheme()); err != nil {
		return nil, err
	}
	return txn, nil
}
