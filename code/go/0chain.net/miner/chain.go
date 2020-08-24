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
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"

	. "0chain.net/core/logging"
	"go.uber.org/zap"
)

//RoundMismatch - to indicate an error where the current round and the given round don't match
const RoundMismatch = "round_mismatch"

//ErrRoundMismatch - an error object for mismatched round error
var ErrRoundMismatch = common.NewError(RoundMismatch, "Current round number of the chain doesn't match the block generation round")

//RRSMismatch -- to indicate an error when the current round RRS is different than the block that is generated. Typically happens when timeout count changes while a block is being made
const RRSMismatch = "rrs_mismatch"

//ErrRRSMismatch - and error when rrs mismatch happens
var ErrRRSMismatch = common.NewError(RRSMismatch, "RRS for current round of the chain doesn't match the block rrs")

//RoundTimeout - to indicate an error where the round timeout has happened
const RoundTimeout = "round_timeout"

//ErrRoundTimeout - an error object for round timeout error
var ErrRoundTimeout = common.NewError(RoundTimeout, "round timed out")

var (
	minerChain = &Chain{}
)

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	minerChain.Chain = c
	minerChain.blockMessageChannel = make(chan *BlockMessage, 128)
	minerChain.muDKG = &sync.RWMutex{}
	minerChain.roundDkg = round.NewRoundStartingStorage()
	c.SetFetchedNotarizedBlockHandler(minerChain)
	c.RoundF = MinerRoundFactory{}
	mv.viewChangeProcess.init(minerChain)
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

//CreateRoundF this returns an interface{} of type *miner.Round
func (mrf MinerRoundFactory) CreateRoundF(roundNum int64) interface{} {
	mc := GetMinerChain()
	r := round.NewRound(roundNum)
	mr := mc.CreateRound(r)
	mc.AddRound(mr)

	return r
}

/*Chain - A miner chain to manage the miner activities */
type Chain struct {
	*chain.Chain
	blockMessageChannel chan *BlockMessage
	muDKG               *sync.RWMutex
	roundDkg            round.RoundStorage
	discoverClients     bool
	started             uint32

	// view change process control
	viewChangeProcess
}

// SetDiscoverClients set the discover clients parameter
func (mc *Chain) SetDiscoverClients(b bool) {
	mc.discoverClients = b
}

/*GetBlockMessageChannel - get the block messages channel */
func (mc *Chain) GetBlockMessageChannel() chan *BlockMessage {
	return mc.blockMessageChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
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

/*CreateRound - create a round */
func (mc *Chain) CreateRound(r *round.Round) *Round {
	var mr Round
	mr.Round = r
	mr.blocksToVerifyChannel = make(chan *block.Block, mc.NumGenerators)
	mr.verificationTickets = make(map[string]*block.BlockVerificationTicket)
	return &mr
}

/*SetLatestFinalizedBlock - Set latest finalized block */
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

/*SetPreviousBlock - set the previous block */
func (mc *Chain) SetPreviousBlock(ctx context.Context, r round.RoundI, b *block.Block, pb *block.Block) {
	b.SetPreviousBlock(pb)
	b.SetRoundRandomSeed(r.GetRandomSeed())
	mc.SetRoundRank(r, b)
	b.ComputeChainWeight()
}

//GetMinerRound - get the miner's version of the round
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

/*SaveClients - save clients from the block */
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

// isViewChanging is period from, for example, 501 to 503 blocks inclusive.
func (mc *Chain) isViewChanging(round int64) (is bool) {

	if !config.DevConfiguration.ViewChange {
		return
	}

	// don't offset the round, offset the next VC round and
	// DKG starting round instead (below)

	var (
		cdkg = mc.GetCurrentDKG(round)

		nvc    = mbRoundOffset(mc.GetNextViewChange()) // add MB offset
		cdkgsr = mbRoundOffset(cdkg.StartingRound)     // add MB offset
	)

	// VC process consist of 3 rounds: 501, 502 and 503 (for example)
	// - 501 and 502 -- VC preparation
	// - 503         -- the view change in person

	// 503 is view change round
	// 502 and 501 are VC preparation rounds

	is = (cdkg == nil || cdkgsr <= nvc) &&
		((nvc-2 == round) || (nvc-1 == round) || (nvc == round))
	return
}

func (mc *Chain) isNeedViewChange(round int64) (is bool) {

	if !config.DevConfiguration.ViewChange {
		return
	}

	// don't offset the round, offset the next VC round and
	// DKG starting round instead (below)

	var (
		cdkg = mc.GetCurrentDKG(round)

		nvc    = mbRoundOffset(mc.GetNextViewChange()) // add MB offset
		cdkgsr = mbRoundOffset(cdkg.StartingRound)     // add MB offset
	)

	is = (nvc == round) && (cdkg == nil || cdkgsr <= nvc)
	return
}

func (mc *Chain) ViewChange(ctx context.Context, nRound int64) (
	ok bool, err error) {

	if !mc.isNeedViewChange(nRound) {
		return false, nil
	}

	viewChangeMutex.Lock()
	defer viewChangeMutex.Unlock()

	var (
		vcmb = mc.GetViewChangeMagicBlock(nRound)
		mb   = mc.GetMagicBlock(nRound)
	)

	if vcmb != nil {
		if mb == nil || mb.MagicBlockNumber != vcmb.MagicBlockNumber {
			if err = mc.UpdateMagicBlock(vcmb); err != nil {
				Logger.DPanic(err.Error())
			}
		}

		mc.UpdateNodesFromMagicBlock(vcmb)

		mc.SetNextViewChange(0)
		go mc.PruneRoundStorage(ctx, mc.getPruneCountRoundStorage(),
			mc.roundDkg, mc.MagicBlockStorage)

		return true, nil
	}

	if err = mc.SetDKGSFromStore(ctx, mb); err != nil {
		Logger.DPanic(err.Error())
	}
	return true, nil
}

// TODO (sfxdx): TO REMOVE
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
	if rn < 3 {
		return rn // the same
	}
	return rn + 2 // MB offset
}

// GetCurrentDKG returns DKG by round number
func (mc *Chain) GetCurrentDKG(round int64) *bls.DKG {

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
