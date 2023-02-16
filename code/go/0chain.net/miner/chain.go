package miner

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/client"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/cache"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"github.com/0chain/common/core/logging"
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

	mcGuard sync.RWMutex
)

/*SetupMinerChain - setup the miner's chain */
func SetupMinerChain(c *chain.Chain) {
	mcGuard.Lock()
	defer mcGuard.Unlock()

	minerChain.Chain = c
	minerChain.Chain.OnBlockAdded = func(b *block.Block) {
	}
	minerChain.ChainConfig = c.ChainConfig

	minerChain.blockMessageChannel = make(chan *BlockMessage, 128)
	minerChain.muDKG = &sync.RWMutex{}
	minerChain.roundDkg = round.NewRoundStartingStorage()
	c.SetFetchedNotarizedBlockHandler(minerChain)
	c.SetViewChanger(minerChain)
	c.RoundF = MinerRoundFactory{}
	// view change / DKG
	minerChain.viewChangeProcess.init(minerChain)
	// restart round event
	minerChain.subRestartRoundEventChannel = make(chan chan struct{})
	minerChain.unsubRestartRoundEventChannel = make(chan chan struct{})
	minerChain.restartRoundEventChannel = make(chan struct{})
	minerChain.restartRoundEventWorkerIsDoneChannel = make(chan struct{})
	minerChain.nbpMutex = &sync.Mutex{}
	minerChain.notarizationBlockProcessMap = make(map[string]struct{})
	minerChain.notarizationBlockProcessC = make(chan *Notarization, 10)
	minerChain.blockVerifyC = make(chan *block.Block, 10) // the channel buffer size need to be adjusted
	minerChain.validateTxnsWithContext = common.NewWithContextFunc(1)
	minerChain.notarizingBlocksTasks = make(map[string]chan struct{})
	minerChain.notarizingBlocksResults = cache.NewLRUCache[string, bool](1000)
	minerChain.nbmMutex = &sync.Mutex{}
	minerChain.verifyBlockNotarizationWorker = common.NewWithContextFunc(4)
	minerChain.mergeBlockVRFSharesWorker = common.NewWithContextFunc(1)
	minerChain.verifyCachedVRFSharesWorker = common.NewWithContextFunc(1)
	minerChain.generateBlockWorker = common.NewWithContextFunc(1)
}

/*GetMinerChain - get the miner's chain */
func GetMinerChain() *Chain {
	mcGuard.RLock()
	defer mcGuard.RUnlock()
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
func (mrf MinerRoundFactory) CreateRoundF(roundNum int64) round.RoundI {
	mc := GetMinerChain()
	r := round.NewRound(roundNum)
	return mc.CreateRound(r)
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

	// restart round event (rre)
	subRestartRoundEventChannel          chan chan struct{} // subscribe for rre
	unsubRestartRoundEventChannel        chan chan struct{} // unsubscribe rre
	restartRoundEventChannel             chan struct{}      // trigger rre
	restartRoundEventWorkerIsDoneChannel chan struct{}      // rre worker closed
	nbpMutex                             *sync.Mutex
	notarizationBlockProcessMap          map[string]struct{}
	notarizationBlockProcessC            chan *Notarization
	blockVerifyC                         chan *block.Block
	validateTxnsWithContext              *common.WithContextFunc
	notarizingBlocksTasks                map[string]chan struct{}
	notarizingBlocksResults              *cache.LRU[string, bool]
	nbmMutex                             *sync.Mutex
	verifyBlockNotarizationWorker        *common.WithContextFunc
	mergeBlockVRFSharesWorker            *common.WithContextFunc
	verifyCachedVRFSharesWorker          *common.WithContextFunc
	generateBlockWorker                  *common.WithContextFunc
}

func (mc *Chain) sendRestartRoundEvent(ctx context.Context) {
	select {
	case <-ctx.Done(): // caller context is done
	case <-mc.restartRoundEventWorkerIsDoneChannel: // worker context is done
	case mc.restartRoundEventChannel <- struct{}{}:
	}
}

func (mc *Chain) subRestartRoundEvent() (subq chan struct{}) {
	subq = make(chan struct{}, 1)
	select {
	case <-mc.restartRoundEventWorkerIsDoneChannel: // worker context is done
	case mc.subRestartRoundEventChannel <- subq:
	}
	return
}

func (mc *Chain) unsubRestartRoundEvent(subq chan struct{}) {
	select {
	case <-mc.restartRoundEventWorkerIsDoneChannel: // worker context is done
	case mc.unsubRestartRoundEventChannel <- subq:
	}
}

// SetDiscoverClients set the discover clients parameter
func (mc *Chain) SetDiscoverClients(b bool) {
	mc.discoverClients = b
}

// PushBlockMessageChannel pushes the block message to the process channel
func (mc *Chain) PushBlockMessageChannel(bm *BlockMessage) {
	go func() {
		select {
		case mc.blockMessageChannel <- bm:
		case <-time.After(3 * time.Second):
			logging.Logger.Warn("push block message to channel timeout",
				zap.Int("message type", bm.Type))
		}
	}()
}

// SetupGenesisBlock - setup the genesis block for this chain.
func (mc *Chain) SetupGenesisBlock(hash string, magicBlock *block.MagicBlock, initStates *state.InitStates) *block.Block {
	gr, gb := mc.GenerateGenesisBlock(hash, magicBlock, initStates)
	rr, ok := gr.(*round.Round)
	if !ok {
		panic("Genesis round cannot convert to *round.Round")
	}
	mgr := mc.CreateRound(rr)
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
	mr.blocksToVerifyChannel = make(chan *block.Block, mc.GetGeneratorsNumOfRound(r.GetRoundNumber()))
	mr.verificationTickets = make(map[string]*block.BlockVerificationTicket)
	mr.vrfSharesCache = newVRFSharesCache()
	return &mr
}

// SetLatestFinalizedBlock - sets the latest finalized block.
func (mc *Chain) SetLatestFinalizedBlock(ctx context.Context, b *block.Block) {
	var r = round.NewRound(b.Round)
	mr := mc.CreateRound(r)
	mr = mc.AddRound(mr).(*Round)
	mc.SetRandomSeed(mr, b.GetRoundRandomSeed())
	mc.AddRoundBlock(mr, b)
	mc.AddNotarizedBlock(mr, b)
	mc.Chain.SetLatestFinalizedBlock(b)
	if b.IsStateComputed() {
		if err := mc.SaveChanges(ctx, b); err != nil {
			logging.Logger.Error("set lfb save changes failed",
				zap.Error(err),
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash))
		}
	}
}

func (mc *Chain) deleteTxns(txns []datastore.Entity) error {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), transactionMetadataProvider)
	defer memorystore.Close(ctx)
	return transactionMetadataProvider.GetStore().MultiDelete(ctx, transactionMetadataProvider, txns)
}

// SetPreviousBlock - set the previous block.
func (mc *Chain) SetPreviousBlock(r round.RoundI, b *block.Block, pb *block.Block) {
	b.SetPreviousBlock(pb)
	mc.SetRoundRank(r, b)
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
func (mc *Chain) SaveClients(clients []*client.Client) error {
	var err error
	clientKeys := make([]datastore.Key, len(clients))
	for idx, c := range clients {
		clientKeys[idx] = c.GetKey()
	}
	clientEntityMetadata := datastore.GetEntityMetadata("client")
	cEntities := datastore.AllocateEntities(len(clients), clientEntityMetadata)
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), clientEntityMetadata)
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

// ViewChange on finalized (!) block. Miners check magic blocks during
// generation and notarization. A finalized block should be trusted.
func (mc *Chain) ViewChange(ctx context.Context, b *block.Block) (err error) {
	if !mc.ChainConfig.IsViewChangeEnabled() {
		return
	}

	var (
		mb  = b.MagicBlock
		nvc int64
	)

	if nvc, err = mc.NextViewChangeOfBlock(b); err != nil {
		return common.NewErrorf("view_change", "getting nvc: %v", err)
	}

	// set / update the next view change of protocol view change (RAM)
	//
	// note: this approach works where a miners is active and finalizes blocks
	//       but for inactive miners we have to set next view change based on
	//       blocks fetched from sharders
	mc.SetNextViewChange(nvc)

	// next view change is expected, but not given;
	// it means the MB rejected by Miner SC
	if b.Round == nvc && mb == nil {
		return // no MB no VC
	}

	// just skip the block if it hasn't a MB
	if mb == nil {
		return // no MB, no VC
	}

	// view change

	if err = mc.UpdateMagicBlock(mb); err != nil {
		return common.NewErrorf("view_change", "updating MB: %v", err)
	}

	mc.SetLatestFinalizedMagicBlock(b)

	go mc.PruneRoundStorage(mc.getPruneCountRoundStorage(),
		mc.roundDkg, mc.MagicBlockStorage)

	// set DKG if this node is miner of new MB (it have to have the DKG)
	var selfNodeKey = node.Self.Underlying().GetKey()

	if !mb.Miners.HasNode(selfNodeKey) {
		return // ok, all done
	}

	// this must be ok, if not -- return error
	if err = mc.SetDKGSFromStore(ctx, mb); err != nil {
		logging.Logger.Error("view_change - set DKG failed",
			zap.Int64("mb_starting_round", mb.StartingRound),
			zap.Error(err))
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
	//  - 504 - generate round (new MB/DKG can be used, but slower, use old)
	//  - 505 - generate block (use new MB/DKG)

	return
}

func StartChainRequestHandler(_ context.Context, req *http.Request) (interface{}, error) {
	nodeID := req.Header.Get(node.HeaderNodeID)
	mc := GetMinerChain()

	r, err := strconv.Atoi(req.FormValue("round"))
	if err != nil {
		logging.Logger.Error("failed to send start chain", zap.Error(err))
		return nil, err
	}

	mb := mc.GetMagicBlock(int64(r))
	if mb == nil || !mb.Miners.HasNode(nodeID) {
		logging.Logger.Error("failed to send start chain", zap.String("id", nodeID))
		return nil, common.NewError("failed to send start chain", "miner is not in active set")
	}

	if mc.GetCurrentRound() != int64(r) {
		logging.Logger.Error("failed to send start chain -- different rounds", zap.Int64("current_round", mc.GetCurrentRound()), zap.Int("requested_round", r))
		return nil, common.NewError("failed to send start chain", fmt.Sprintf("differt_rounds -- current_round: %v, requested_round: %v", mc.GetCurrentRound(), r))
	}
	message := datastore.GetEntityMetadata("start_chain").Instance().(*StartChain)
	message.Start = !mc.isStarted()
	message.ID = req.FormValue("round")
	return message, nil
}

func (mc *Chain) SetStarted() {
	if !atomic.CompareAndSwapUint32(&mc.started, 0, 1) {
		logging.Logger.Warn("chain already started")
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

func (mc *Chain) RejectNotarizedBlock(_ string) bool {
	return false
}
