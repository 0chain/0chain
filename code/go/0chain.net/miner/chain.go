package miner

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

const (
	// RoundMismatch - to indicate an error where the current round and the
	// given round don't match.
	RoundMismatch = "round_mismatch"
	// RRSMismatch -- to indicate an error when the current round RRS is
	// different than the block that is generated. Typically happens when
	// timeout count changes while a block is being made.
	RRSMismatch = "rrs_mismatch"
	// RoundTimeout - to indicate an error where the round timeout has happened.
	RoundTimeout = "round_timeout"
)

var (
	// ErrRoundMismatch - an error object for mismatched round error.
	ErrRoundMismatch = common.NewError(RoundMismatch, "Current round number"+
		" of the chain doesn't match the block generation round")
	// ErrRRSMismatch - and error when rrs mismatch happens.
	ErrRRSMismatch = common.NewError(RRSMismatch, "RRS for current round"+
		" of the chain doesn't match the block rrs")
	// ErrRoundTimeout - an error object for round timeout error.
	ErrRoundTimeout = common.NewError(RoundTimeout, "round timed out")

	minerChain = &Chain{}
)

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = c
	minerChain.blockMessageChannel = make(chan *BlockMessage, 128)
	minerChain.muDKG = &sync.RWMutex{}
	minerChain.roundDkg = round.NewRoundStartingStorage()
	c.SetFetchedNotarizedBlockHandler(minerChain)
	c.SetViewChanger(minerChain)
	c.RoundF = MinerRoundFactory{}
	minerChain.viewChangeProcess.init(minerChain)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	return minerChain
}

type StartChain struct {
	datastore.IDField
	Start bool
}

var startChainEntityMetadata *datastore.EntityMetadataImpl

func (sc *StartChain) GetEntityMetadata() datastore.EntityMetadata {
	return startChainEntityMetadata
}

func StartChainProvider() datastore.Entity {
	sc := &StartChain{}
	return sc
}

func SetupStartChainEntity() {
	startChainEntityMetadata = datastore.MetadataProvider()
	startChainEntityMetadata.Name = "start_chain"
	startChainEntityMetadata.Provider = StartChainProvider
	startChainEntityMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("start_chain", startChainEntityMetadata)
}

// MinerRoundFactory -
type MinerRoundFactory struct{}

// CreateRoundF this returns an interface{} of type *miner.Round
func (mrf MinerRoundFactory) CreateRoundF(roundNum int64) interface{} {
	mc := GetMinerChain()
	r := round.NewRound(roundNum)
	mr := mc.CreateRound(r)
	mc.AddRound(mr)

	return r
}

// Chain - a miner chain to manage the miner activities.
type Chain struct {
	*chain.Chain
	blockMessageChannel chan *BlockMessage
	muDKG               *sync.RWMutex
	roundDkg            round.RoundStorage
	discoverClients     bool
	started             uint32

	// view change process control
	viewChangeProcess

	// not. blocks pulling joining at VC
	pullingPin int64
}

func (mc *Chain) startPulling() (ok bool) {
	return atomic.CompareAndSwapInt64(&mc.pullingPin, 0, 1)
}

func (mc *Chain) stopPulling() (ok bool) {
	return atomic.CompareAndSwapInt64(&mc.pullingPin, 1, 0)
}

// SetDiscoverClients set the discover clients parameter
func (mc *Chain) SetDiscoverClients(b bool) {
	mc.discoverClients = b
}

// GetBlockMessageChannel - get the block messages channel.
func (mc *Chain) GetBlockMessageChannel() chan *BlockMessage {
	return mc.blockMessageChannel
}

// SetupGenesisBlock - setup the genesis block for this chain.
func (mc *Chain) SetupGenesisBlock(hash string, magicBlock *block.MagicBlock) *block.Block {
	gr, gb := mc.GenerateGenesisBlock(hash, magicBlock)
	if gr == nil || gb == nil {
		panic("Genesis round/block can't be null")
	}
	rr, ok := gr.(*round.Round)
	if !ok {
		panic("Genesis round cannot convert to *round.Round")
	}
	mgr := mc.CreateRound(rr)
	mgr.ComputeMinerRanks(gb.MagicBlock.Miners)
	mc.AddRound(mgr)
	mc.AddGenesisBlock(gb)
	for _, sharder := range gb.Sharders.Nodes {
		sharder.SetStatus(node.NodeStatusInactive)
	}
	return gb
}

// CreateRound - create a round.
func (mc *Chain) CreateRound(r *round.Round) *Round {
	var mr Round
	mr.Round = r
	mr.blocksToVerifyChannel = make(chan *block.Block, mc.NumGenerators)
	mr.verificationTickets = make(map[string]*block.BlockVerificationTicket)
	return &mr
}

// SetLatestFinalizedBlock - sets the latest finalized block.
func (mc *Chain) SetLatestFinalizedBlock(ctx context.Context, b *block.Block) {
	var r = round.NewRound(b.Round)
	mr := mc.CreateRound(r)
	mr = mc.AddRound(mr).(*Round)
	mc.SetRandomSeed(mr, b.GetRoundRandomSeed())
	mc.AddRoundBlock(mr, b)
	mc.AddNotarizedBlock(ctx, mr, b)
	mc.Chain.SetLatestFinalizedBlock(b)
}

func (mc *Chain) deleteTxns(txns []datastore.Entity) error {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), transactionMetadataProvider)
	defer memorystore.Close(ctx)
	return transactionMetadataProvider.GetStore().MultiDelete(ctx, transactionMetadataProvider, txns)
}

// SetPreviousBlock - set the previous block.
func (mc *Chain) SetPreviousBlock(ctx context.Context, r round.RoundI,
	b *block.Block, pb *block.Block) {

	b.SetPreviousBlock(pb)
	b.SetRoundRandomSeed(r.GetRandomSeed())
	mc.SetRoundRank(r, b)
	b.ComputeChainWeight()
}

// GetMinerRound - get the miner's version of the round.
func (mc *Chain) GetMinerRound(roundNumber int64) *Round {
	r := mc.GetRound(roundNumber)
	if r == nil {
		return nil
	}
	mr, ok := r.(*Round)
	if !ok {
		return nil
	}
	return mr
}

// SaveClients - save clients from the block.
func (mc *Chain) SaveClients(ctx context.Context, clients []*client.Client) error {
	var err error
	clientKeys := make([]datastore.Key, len(clients))
	for idx, c := range clients {
		clientKeys[idx] = c.GetKey()
	}
	clientEntityMetadata := datastore.GetEntityMetadata("client")
	cEntities := datastore.AllocateEntities(len(clients), clientEntityMetadata)
	ctx = memorystore.WithEntityConnection(common.GetRootContext(), clientEntityMetadata)
	defer memorystore.Close(ctx)
	err = clientEntityMetadata.GetStore().MultiRead(ctx, clientEntityMetadata, clientKeys, cEntities)
	if err != nil {
		return err
	}
	ctx = datastore.WithAsyncChannel(ctx, client.ClientEntityChannel)
	for idx, c := range clients {
		if !datastore.IsEmpty(cEntities[idx].GetKey()) {
			continue
		}
		_, cerr := client.PutClient(ctx, c)
		if cerr != nil {
			err = cerr
		}
	}
	return err
}

// TODO (sfxdx): TO REMOVE (old code)
//
// func (mc *Chain) isNeedViewChange(round int64) (is bool) {
//
// 	if !config.DevConfiguration.ViewChange {
// 		return
// 	}
//
// 	// don't offset anything here, we have to do VC when round/block
// 	// is finalized and thus we are using rounds without any offsets
//
// 	var (
// 		cdkg = mc.GetDKG(round)
//
// 		nvc    = mc.NextViewChange() // no MB offset
// 		cdkgsr = cdkg.StartingRound  // no MB offset
// 	)
//
// 	is = (nvc == round) && (cdkg == nil || cdkgsr <= nvc)
// 	return
// }

// isViewChanging returns true for 501-504 rounds
func (mc *Chain) isViewChanging(round int64) (is bool) {
	var (
		lfb = mc.GetLatestFinalizedBlock() //
		nvc = mc.NextViewChange(lfb)       // expected
	)
	return round >= nvc && round < nvc+chain.ViewChangeOffset
}

// The sJoining returns true if this miner joins blockchain on current view
// changing. For rounds, for example, from 501 to 504.
func (mc *Chain) isJoining(rn int64) (is bool) {

	var (
		lfb = mc.GetLatestFinalizedBlock()
		nvc = mc.NextViewChange(lfb)
	)

	if lfb.Round >= nvc && rn < nvc+chain.ViewChangeOffset {
		// get current magic block
		var (
			mb  = mc.GetMagicBlock(rn)
			nmb = mc.GetMagicBlock(nvc + chain.ViewChangeOffset)

			key = node.Self.Underlying().GetKey()
		)
		// TODO (sfxdx): remove the logs
		if nmb.StartingRound != nvc {
			println("isJoining: unexpected next MB", "MB", mb.StartingRound, "NMB", nmb.StartingRound)
			return false // unexpected next magic block
		}
		// TODO (sfxdx): remove the logs
		if !mb.Miners.HasNode(key) && nmb.Miners.HasNode(key) {
			println("VIEW CHANGING NODE START NEW ROUND")
			println("  MB", mb.StartingRound, "NMB", nmb.StartingRound)
			return true
		}
	}

	return // false
}

// TODO (sfdx): TO REMOVE OR TO KEEP OR TO MODIFY (questionable code)
//
//
// // a node leaving BC on VC coming have to finalize its last round; since it
// // can't start next round (504) and finalize 503 for 502 block; it returns
// // true for 502 round only (the first round of new MB/DKG);
// func (mc *Chain) isLeaving(round int64) (is bool) {
// 	if !mc.isViewChanging(round) {
// 		return // if not view changing, then not leaving (short circuit)
// 	}
// 	var mb, nmb = mc.GetMagicBlock(round), mc.GetMagicBlock(round + 1)
// 	if mb.StartingRound >= nmb.StartingRound {
// 		return // not leaving
// 	}
// 	var nmbsroff = mbRoundOffset(nmb.StartingRound)
// 	return round == nmbsroff // is
// }

// TODO (sfxdx): TO REMOVE (draft code)
//
// // in goroutine
// func (mc *Chain) kickNewMiners(ctx context.Context, b *block.Block) {
//
// 	if node.Self.Underlying().Type != node.NodeTypeMiner {
// 		return // only miner should do that
// 	}
//
// 	// get previous magic block
// 	var prev = mc.GetMagicBlock(b.Round)
// 	if prev.MagicBlockNumber != b.MagicBlock.MagicBlockNumber-1 {
// 		Logger.Error("unexpected previous magic block",
// 			zap.Int64("mb_number", b.MagicBlock.MagicBlockNumber),
// 			zap.Int64("prev_mb_number", prev.MagicBlockNumber))
// 		return
// 	}
//
// 	// TODO (sfxd): kick the new miners, force them to pull LFB from sharders
// }

// ViewChange on finalized (!) block. Miners check magic blocks during
// generation and notarization. A finalized block should be trusted.
func (mc *Chain) ViewChange(ctx context.Context, b *block.Block) (err error) {

	var (
		lfb = mc.GetLatestFinalizedBlock()
		mb  = b.MagicBlock
		nvc = mc.NextViewChange(lfb)
	)

	// next view change is expected, but not given; it means the MB rejected
	// by Miner SC (managed by miner SC: and we have to reset NextViewChange
	// to previous MB for block generation and verification process);
	if b.Round == nvc && mb == nil {
		// var mbx = mc.GetMagicBlock(b.Round)
		// mc.SetNextViewChange(mbx.StartingRound)
		println("MINER VC: NO MB, NO VC", b.Round, "BUT EXPECTED")
		return // no MB no VC
	}

	// just skip the block if it hasn't a MB
	if mb == nil {
		return // no MB, no VC
	}

	println("MINER VIEW CHANGE")

	// view change

	if err = mc.UpdateMagicBlock(mb); err != nil {
		return common.NewErrorf("view_change", "updating MB: %v", err)
	}

	mc.UpdateNodesFromMagicBlock(mb)
	go mc.PruneRoundStorage(ctx, mc.getPruneCountRoundStorage(),
		mc.roundDkg, mc.MagicBlockStorage)

	// set DKG if this node is miner of new MB (it have to have the DKG)
	var selfNodeKey = node.Self.Underlying().GetKey()

	if !mb.Miners.HasNode(selfNodeKey) {
		return // ok, all done
	}

	// this must be ok, if not -- return error
	if err = mc.SetDKGSFromStore(ctx, mb); err != nil {
		return
	}

	// new miners has no previous round, and LFB, this block becomes
	// LFB and new miners have to get it from miners or sharders to
	// join BC; now we have to kick them to don't wait for while and
	// get the block from sharders and join BC; anyway the new miners
	// will pool LFB (501) from sharders and join; but the kick used
	// to skip a waiting

	// to mine 503 round (new round with new nodes and new MB)
	// the new miners need:
	//    - 501 block with MB, corresponding DKG saved locally
	//    - 502 block and round (for previous round random seed)

	// the flow:
	//
	//  - 501 - finalized
	//  - 502 - finalize round (finalize 501 block)
	//  - 503 - verify round blocks
	//  - 504 - generate round (new MB/DKG can be used, but slower)
	//  - 505 - generate block (new MB/DKG)

	return
}

// TODO (sfxdx): TO REMOVE (old code)
//
// // Send a notarized block for new miners
// func (mc *Chain) sendNotarizedBlockToNewMiners(ctx context.Context, nRound int64,
// 	viewChangeMagicBlock, currentMagicBlock *block.MagicBlock) {
// 	prevRound := mc.GetMinerRound(nRound)
// 	prevMinerNodes := mc.GetMagicBlock(nRound).Miners
// 	if prevRound == nil {
// 		Logger.Error("round not found", zap.Any("round", nRound))
// 		return
// 	}
// 	if prevRound.Block == nil {
// 		Logger.Error("block round not found", zap.Any("round", nRound))
// 		return
// 	}
// 	selfID := node.Self.Underlying().GetKey()
// 	if currentMagicBlock.Miners.GetNode(selfID) == nil {
// 		// A miner that was active in the previous VC can send a block
// 		return
// 	}

// 	prevBlock := prevRound.Block
// 	allMinersMB := make([]*node.Node, 0)
// 	for _, n := range viewChangeMagicBlock.Miners.NodesMap {
// 		if selfID == n.GetKey() {
// 			continue
// 		}
// 		if prevMinerNodes.GetNode(n.GetKey()) == nil {
// 			allMinersMB = append(allMinersMB, n)
// 		}
// 	}
// 	if len(allMinersMB) != 0 {
// 		go mc.SendNotarizedBlockToPoolNodes(ctx, prevBlock, viewChangeMagicBlock.Miners,
// 			allMinersMB, chain.DefaultRetrySendNotarizedBlockNewMiner)
// 	}
// }

func (mc *Chain) ChainStarted(ctx context.Context) bool {
	timer := time.NewTimer(time.Second)
	timeoutCount := 0
	for true {
		select {
		case <-ctx.Done():
			return false
		case <-timer.C:
			var start int
			var started int
			mb := mc.GetCurrentMagicBlock()
			for _, n := range mb.Miners.CopyNodesMap() {
				mc.RequestStartChain(n, &start, &started)
			}
			if start >= mb.T {
				return false
			}
			if started >= mb.T {
				return true
			}
			if timeoutCount == 20 || mc.isStarted() {
				return mc.isStarted()
			}
			timeoutCount++
			timer = time.NewTimer(time.Millisecond * time.Duration(mc.RoundTimeoutSofttoMin))
		}
	}
	return false
}

func StartChainRequestHandler(ctx context.Context, r *http.Request) (interface{}, error) {
	nodeID := r.Header.Get(node.HeaderNodeID)
	mc := GetMinerChain()

	round, err := strconv.Atoi(r.FormValue("round"))
	if err != nil {
		Logger.Error("failed to send start chain", zap.Any("error", err))
		return nil, err
	}

	mb := mc.GetMagicBlock(int64(round))
	if mb == nil || !mb.Miners.HasNode(nodeID) {
		Logger.Error("failed to send start chain", zap.Any("id", nodeID))
		return nil, common.NewError("failed to send start chain", "miner is not in active set")
	}

	if mc.GetCurrentRound() != int64(round) {
		Logger.Error("failed to send start chain -- different rounds", zap.Any("current_round", mc.GetCurrentRound()), zap.Any("requested_round", round))
		return nil, common.NewError("failed to send start chain", fmt.Sprintf("differt_rounds -- current_round: %v, requested_round: %v", mc.GetCurrentRound(), round))
	}
	message := datastore.GetEntityMetadata("start_chain").Instance().(*StartChain)
	message.Start = !mc.isStarted()
	message.ID = r.FormValue("round")
	return message, nil
}

/*SendDKGShare sends the generated secShare to the given node */
func (mc *Chain) RequestStartChain(n *node.Node, start, started *int) error {
	if node.Self.Underlying().GetKey() == n.ID {
		if !mc.isStarted() {
			*start++
		} else {
			*started++
		}
		return nil
	}
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		startChain, ok := entity.(*StartChain)
		if !ok {
			err := common.NewError("invalid object", fmt.Sprintf("entity: %v", entity))
			Logger.Error("failed to request start chain", zap.Any("error", err))
			return nil, err
		}
		if startChain.Start {
			*start++
		} else {
			*started++
		}
		return startChain, nil
	}
	params := &url.Values{}
	params.Add("round", strconv.FormatInt(mc.GetCurrentRound(), 10))
	ctx := common.GetRootContext()
	n.RequestEntityFromNode(ctx, ChainStartSender, params, handler)
	return nil
}

func (mc *Chain) SetStarted() {
	if !atomic.CompareAndSwapUint32(&mc.started, 0, 1) {
		Logger.Warn("chain already started")
	}
}

func (mc *Chain) isStarted() bool {
	return atomic.LoadUint32(&mc.started) == 1
}

// SaveMagicBlock returns nil.
func (mc *Chain) SaveMagicBlock() chain.MagicBlockSaveFunc {
	return nil
}

func mbRoundOffset(rn int64) int64 {
	if rn < chain.ViewChangeOffset+1 {
		return rn // the same
	}
	return rn - chain.ViewChangeOffset // MB offset
}

// GetDKG returns DKG by round number.
func (mc *Chain) GetDKG(round int64) *bls.DKG {

	round = mbRoundOffset(round)

	mc.muDKG.RLock()
	defer mc.muDKG.RUnlock()
	entity := mc.roundDkg.Get(round)
	if entity == nil {
		return nil
	}
	return entity.(*bls.DKG)
}

// SetDKG sets DKG for the start round
func (mc *Chain) SetDKG(dkg *bls.DKG, startingRound int64) error {
	mc.muDKG.Lock()
	defer mc.muDKG.Unlock()
	return mc.roundDkg.Put(dkg, startingRound)
}
