package miner

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/client"
	"0chain.net/chaincore/transaction"
	"go.uber.org/zap"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
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
	minerChain.manualViewChangeC = make(chan *ViewChangeEvent, 1)
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
	discoverClients     bool
	started             uint32

	// view change process control
	viewChangeProcess
	manualViewChangeC chan *ViewChangeEvent // TODO: process the StoreDKG and magic block to DB

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

type ViewChangeEvent struct {
	MagicBlock *block.MagicBlock
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
	b = mc.AddRoundBlock(mr, b)
	// mr.SetFinalized()
	mr.Finalize(b)
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

	// Check if the new LFB round requires a different MB than current finalized MB.
	// This handles the case where forward sync advances LFB past an MB transition point.
	expectedMB := mc.GetMagicBlock(b.Round)
	if expectedMB == nil {
		return
	}
	currentFinalizedMB := mc.GetLatestFinalizedMagicBlock(ctx)
	if currentFinalizedMB == nil || currentFinalizedMB.MagicBlock == nil {
		return
	}
	// Only update to a NEWER MB (higher number), never downgrade
	if expectedMB.MagicBlockNumber > currentFinalizedMB.MagicBlock.MagicBlockNumber {
		logging.Logger.Info("SetLatestFinalizedBlock - LFB crossed MB transition, updating finalized MB",
			zap.Int64("lfb_round", b.Round),
			zap.Int64("old_mb", currentFinalizedMB.MagicBlock.MagicBlockNumber),
			zap.Int64("old_mb_sr", currentFinalizedMB.MagicBlock.StartingRound),
			zap.Int64("new_mb", expectedMB.MagicBlockNumber),
			zap.Int64("new_mb_sr", expectedMB.StartingRound))
		// Create a block wrapper for SetLatestFinalizedMagicBlock
		mbBlock := &block.Block{MagicBlock: expectedMB}
		mbBlock.Round = expectedMB.StartingRound
		mbBlock.Hash = expectedMB.Hash
		mc.SetLatestFinalizedMagicBlock(mbBlock)
	}
}

// LoadLatestBlocksFromStore loads LFB and LFMB from store and sets them
// to corresponding fields of the sharder's Chain.
func (mc *Chain) LoadLatestBlocksFromStore(ctx context.Context) error {
	// var bl *blocksLoaded
	lfbr, err := mc.LoadLFBRound()
	if err != nil {
		logging.Logger.Warn("load_lfb - could not load lfb from state DB, trying to fetch current LFB from sharders",
			zap.Error(err))
		return mc.tryFetchCurrentLFBFromSharders(ctx)
	}

	logging.Logger.Debug("load_lfb - load from stateDB",
		zap.Int64("round", lfbr.Round),
		zap.String("block", lfbr.Hash))

	// fetch from sharders
	// retry 10 times, each time wait for 3-5 seconds with backoff.
	// the main reason for retry is that sharders APIs may not ready yet after all miners/sharders restarted
	retry := 10
	var b *block.Block
	for i := 0; i < retry; i++ {
		b, err = mc.GetNotarizedBlockFromSharders(ctx, lfbr.Hash, lfbr.Round)
		if err != nil {
			waitTime := time.Duration(3+i) * time.Second
			logging.Logger.Warn("load_lfb - could not fetch block from sharders, waiting for retry...",
				zap.Int64("round", lfbr.Round), zap.String("block", lfbr.Hash),
				zap.Int("retry", i+1), zap.Int("max_retries", retry),
				zap.Duration("wait", waitTime), zap.Error(err))
			time.Sleep(waitTime)
			continue
		}
		break
	}

	if b == nil {
		// Stored LFB not available from sharders - try to fetch current LFB from sharders instead
		logging.Logger.Warn("load_lfb - stored LFB not available from sharders, trying to fetch current LFB",
			zap.Int64("stored_round", lfbr.Round),
			zap.String("stored_hash", lfbr.Hash))
		return mc.tryFetchCurrentLFBFromSharders(ctx)
	}

	// Verify block has sufficient verification tickets before accepting
	if err = mc.VerifyBlockNotarization(ctx, b); err != nil {
		logging.Logger.Error("load_lfb - block notarization verification failed, falling back to sharder sync",
			zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Int("tickets", len(b.GetVerificationTickets())))
		// Block doesn't have enough tickets - fall back to fetching from sharders
		// which will find blocks with valid notarization
		return mc.tryFetchCurrentLFBFromSharders(ctx)
	}

	// Verify block continuity and finalization depth from sharders
	recommendedLFB, cerr := mc.verifyBlockContinuityFromSharders(ctx, b.Round, 500, 3)
	if cerr != nil {
		logging.Logger.Warn("load_lfb - continuity check failed, falling back to sharder sync",
			zap.Int64("round", b.Round), zap.Error(cerr))
		return mc.tryFetchCurrentLFBFromSharders(ctx)
	}
	if recommendedLFB < b.Round {
		logging.Logger.Info("load_lfb - finalization depth requires earlier LFB",
			zap.Int64("stored_lfb", b.Round),
			zap.Int64("recommended_lfb", recommendedLFB))
		rb, rerr := mc.GetNotarizedBlockFromSharders(ctx, "", recommendedLFB)
		if rerr != nil || rb == nil {
			logging.Logger.Warn("load_lfb - could not fetch recommended LFB, falling back",
				zap.Int64("round", recommendedLFB))
			return mc.tryFetchCurrentLFBFromSharders(ctx)
		}
		b = rb
	}

	b.SetStateStatus(block.StateSuccessful)
	if err = mc.InitBlockState(b); err != nil {
		b.SetStateStatus(0)
		logging.Logger.Error("load_lfb -- can't initialize stored block state",
			zap.Error(err))
		return fmt.Errorf("can't init block state: %v", err) // fatal
	}

	mc.SetLatestFinalizedBlock(ctx, b)

	// Set current round to LFB round (handles both forward sync and rollback)
	if b.Round != mc.GetCurrentRound() {
		mc.SetCurrentRound(b.Round)
	}

	logging.Logger.Info("load_lfb setup LFB from store",
		zap.String("block", b.Hash),
		zap.Int64("round", b.Round),
		zap.Int64("lf_round", mc.GetLatestFinalizedBlock().Round))

	// Check if sharders have a higher LFB - if so, sync to catch up
	// This handles the case where miner restarted and sharders advanced
	fbs := mc.GetLatestFinalizedBlockFromSharderNoFilter(ctx)
	if len(fbs) > 0 && fbs[0].Block != nil && fbs[0].Block.Round > b.Round {
		sharderLFB := fbs[0].Block
		logging.Logger.Info("load_lfb - sharders have higher LFB, syncing forward",
			zap.Int64("local_lfb", b.Round),
			zap.Int64("sharder_lfb", sharderLFB.Round))

		// Sync blocks from our LFB+1 to sharder's LFB
		for r := b.Round + 1; r <= sharderLFB.Round; r++ {
			fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			syncBlock, err := mc.GetNotarizedBlockFromSharders(fetchCtx, "", r)
			cancel()
			if err != nil || syncBlock == nil {
				logging.Logger.Warn("load_lfb - failed to sync block from sharders",
					zap.Int64("round", r), zap.Error(err))
				break
			}
			if err := mc.VerifyBlockNotarization(ctx, syncBlock); err != nil {
				logging.Logger.Warn("load_lfb - sync block failed notarization",
					zap.Int64("round", r), zap.Error(err))
				break
			}
			// Push to block processor to finalize
			if err := mc.PushToBlockProcessor(syncBlock); err != nil {
				logging.Logger.Warn("load_lfb - failed to push sync block",
					zap.Int64("round", r), zap.Error(err))
			}
		}
	}

	// Reset LFB ticket to match actual LFB after any rollback during startup.
	// This ensures the ticket is not set too high from stale network data.
	mc.Chain.ResetLFBTicket(ctx, b)

	// Mark LFB loading as complete - workers can now call BumpLFBTicket
	// and LFBTicketHandler can accept network tickets
	mc.Chain.SetLFBLoadingComplete()

	return nil
}

// verifyBlockContinuityFromSharders fetches blocks from sharders around the
// candidate round, verifies notarization, and returns the recommended LFB round
// based on finalization depth (highest block with finalizationDepth+ notarized successors).
func (mc *Chain) verifyBlockContinuityFromSharders(ctx context.Context, candidateRound int64, targetContinuity int, finalizationDepth int) (int64, error) {
	var ch []*block.Block

	// Fetch blocks backward from candidate
	for r := candidateRound; r > 0 && len(ch) < targetContinuity; r-- {
		fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		b, err := mc.GetNotarizedBlockFromSharders(fetchCtx, "", r)
		cancel()
		if err != nil || b == nil {
			break
		}
		if err := mc.VerifyBlockNotarization(ctx, b); err != nil {
			break
		}
		ch = append([]*block.Block{b}, ch...)
	}

	// Fetch blocks forward from candidate+1
	for r := candidateRound + 1; len(ch) < targetContinuity; r++ {
		fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		b, err := mc.GetNotarizedBlockFromSharders(fetchCtx, "", r)
		cancel()
		if err != nil || b == nil {
			break
		}
		if err := mc.VerifyBlockNotarization(ctx, b); err != nil {
			break
		}
		ch = append(ch, b)
	}

	if len(ch) == 0 {
		return 0, fmt.Errorf("no valid blocks from sharders around round %d", candidateRound)
	}

	logging.Logger.Info("verify_continuity - continuous chain from sharders",
		zap.Int("length", len(ch)),
		zap.Int64("from", ch[0].Round),
		zap.Int64("to", ch[len(ch)-1].Round))

	// Apply finalization depth
	if len(ch) > finalizationDepth {
		idx := len(ch) - 1 - finalizationDepth
		return ch[idx].Round, nil
	}
	return ch[0].Round, nil
}

// tryFetchCurrentLFBFromSharders attempts to fetch the current LFB from sharders
// and set it as the miner's LFB. This is used when local RocksDB state is stale or missing.
// It verifies that blocks have sufficient verification tickets before accepting them.
func (mc *Chain) tryFetchCurrentLFBFromSharders(ctx context.Context) error {
	// Retry fetching from sharders with backoff - sharders may still be starting up
	var fbs []*chain.BlockConsensus
	maxRetries := 10
	for retry := 0; retry < maxRetries; retry++ {
		fbs = mc.GetLatestFinalizedBlockFromSharderNoFilter(ctx)
		if len(fbs) > 0 {
			break
		}
		waitTime := time.Duration(3+retry*2) * time.Second
		logging.Logger.Info("load_lfb - no LFB from sharders yet, waiting for retry",
			zap.Int("retry", retry+1),
			zap.Int("max_retries", maxRetries),
			zap.Duration("wait", waitTime))
		time.Sleep(waitTime)
	}
	if len(fbs) == 0 {
		logging.Logger.Warn("load_lfb - no LFB available from sharders after retries, will use genesis")
		// Mark loading complete even for genesis fallback
		mc.Chain.SetLFBLoadingComplete()
		return nil // Fall back to genesis
	}

	// Find the highest round block that passes notarization verification
	var best *block.Block
	for _, fb := range fbs {
		if fb.Block == nil {
			continue
		}
		// Verify block has sufficient verification tickets
		if err := mc.VerifyBlockNotarization(ctx, fb.Block); err != nil {
			logging.Logger.Debug("load_lfb - block from sharder failed notarization verification",
				zap.Int64("round", fb.Block.Round),
				zap.String("hash", fb.Block.Hash),
				zap.Int("tickets", len(fb.Block.GetVerificationTickets())),
				zap.Error(err))
			continue
		}
		// Block passed verification - check if it's better than current best
		if best == nil || fb.Block.Round > best.Round {
			best = fb.Block
		}
	}

	if best == nil {
		logging.Logger.Warn("load_lfb - no valid LFB with sufficient verification tickets from sharders, will use genesis")
		// Mark loading complete even for genesis fallback
		mc.Chain.SetLFBLoadingComplete()
		return nil // Fall back to genesis
	}

	logging.Logger.Info("load_lfb - fetched current LFB from sharders with valid notarization",
		zap.Int64("round", best.Round),
		zap.String("hash", best.Hash),
		zap.Int("tickets", len(best.GetVerificationTickets())))

	// Verify continuity and finalization depth
	recommendedLFB, cerr := mc.verifyBlockContinuityFromSharders(ctx, best.Round, 500, 3)
	if cerr == nil && recommendedLFB < best.Round {
		logging.Logger.Info("load_lfb - adjusting LFB for finalization depth",
			zap.Int64("sharder_lfb", best.Round),
			zap.Int64("recommended_lfb", recommendedLFB))
		rb, rerr := mc.GetNotarizedBlockFromSharders(ctx, "", recommendedLFB)
		if rerr == nil && rb != nil {
			if verr := mc.VerifyBlockNotarization(ctx, rb); verr == nil {
				best = rb
			}
		}
	}

	// Try to initialize the block's state from local RocksDB
	best.SetStateStatus(block.StateSuccessful)
	if err := mc.InitBlockState(best); err != nil {
		best.SetStateStatus(0)
		// State init failed - this happens when RocksDB is empty/cleared
		// Blockchain state is cumulative - we can't just sync a single block's state
		// Must start from genesis and replay blocks to rebuild state
		logging.Logger.Warn("load_lfb - can't initialize LFB state (local state DB empty?), "+
			"will start from genesis and sync incrementally",
			zap.Int64("network_lfb_round", best.Round),
			zap.String("network_lfb_hash", best.Hash),
			zap.Error(err))
		// Mark loading complete even for genesis fallback
		mc.Chain.SetLFBLoadingComplete()
		return nil // Fall back to genesis - state will be built as blocks are synced
	}

	mc.SetLatestFinalizedBlock(ctx, best)

	// Set current round to LFB round (handles both forward sync and rollback)
	if best.Round != mc.GetCurrentRound() {
		mc.SetCurrentRound(best.Round)
	}

	logging.Logger.Info("load_lfb - successfully set LFB from sharders",
		zap.Int64("round", best.Round),
		zap.String("hash", best.Hash))

	// Reset LFB ticket to match actual LFB after loading from sharders.
	// This ensures the ticket is not set too high from stale network data.
	mc.Chain.ResetLFBTicket(ctx, best)

	// Mark LFB loading as complete - workers can now call BumpLFBTicket
	// and LFBTicketHandler can accept network tickets
	mc.Chain.SetLFBLoadingComplete()

	return nil
}

// ResyncLFBFromSharders re-queries sharders for their current LFB and adopts it.
// This is called when the miner is stuck trying to sync blocks that don't exist.
func (mc *Chain) ResyncLFBFromSharders(ctx context.Context) error {
	// Query sharders for their current LFB
	fbs := mc.GetLatestFinalizedBlockFromSharderNoFilter(ctx)
	if len(fbs) == 0 {
		return fmt.Errorf("no LFB available from sharders")
	}

	// Find the highest round block that passes notarization verification
	var best *block.Block
	for _, fb := range fbs {
		if fb.Block == nil {
			continue
		}
		if err := mc.VerifyBlockNotarization(ctx, fb.Block); err != nil {
			continue
		}
		if best == nil || fb.Block.Round > best.Round {
			best = fb.Block
		}
	}

	if best == nil {
		return fmt.Errorf("no valid LFB with sufficient verification tickets from sharders")
	}

	currentLFB := mc.GetLatestFinalizedBlock()
	if best.Round > currentLFB.Round {
		// Sharder LFB is ahead - sync blocks from local LFB+1 to sharder LFB
		logging.Logger.Info("resync_lfb - sharder LFB ahead, syncing forward",
			zap.Int64("sharder_lfb", best.Round),
			zap.Int64("current_lfb", currentLFB.Round))

		// Sync blocks one by one from sharders
		for r := currentLFB.Round + 1; r <= best.Round; r++ {
			fetchCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
			syncBlock, err := mc.GetNotarizedBlockFromSharders(fetchCtx, "", r)
			cancel()
			if err != nil || syncBlock == nil {
				logging.Logger.Warn("resync_lfb - failed to fetch block from sharders",
					zap.Int64("round", r), zap.Error(err))
				break
			}
			if err := mc.VerifyBlockNotarization(ctx, syncBlock); err != nil {
				logging.Logger.Warn("resync_lfb - block failed notarization verification",
					zap.Int64("round", r), zap.Error(err))
				break
			}
			// Push to block processor for finalization
			if err := mc.PushToBlockProcessor(syncBlock); err != nil {
				logging.Logger.Warn("resync_lfb - failed to push block to processor",
					zap.Int64("round", r), zap.Error(err))
			}
		}
		return nil
	}

	if best.Round == currentLFB.Round {
		// LFB matches - just reset ticket
		logging.Logger.Info("resync_lfb - sharder LFB matches local LFB",
			zap.Int64("lfb_round", best.Round))
		mc.ResetLFBTicket(ctx, currentLFB)
		return nil
	}

	logging.Logger.Info("resync_lfb - adopting lower LFB from sharders",
		zap.Int64("old_lfb", currentLFB.Round),
		zap.Int64("new_lfb", best.Round))

	// Initialize block state
	best.SetStateStatus(block.StateSuccessful)
	if err := mc.InitBlockState(best); err != nil {
		best.SetStateStatus(0)
		return fmt.Errorf("can't initialize LFB state: %v", err)
	}

	// Set as new LFB
	mc.SetLatestFinalizedBlock(ctx, best)
	mc.SetCurrentRound(best.Round)

	// Reset ticket to match new LFB
	mc.ResetLFBTicket(ctx, best)

	return nil
}

func (mc *Chain) deleteTxns(txns []datastore.Entity) error {
	transactionMetadataProvider := datastore.GetEntityMetadata("txn")
	ctx := memorystore.WithEntityConnection(common.GetRootContext(), transactionMetadataProvider)
	defer memorystore.Close(ctx)
	txnHashes := make([]string, len(txns))
	for i, txn := range txns {
		txnHashes[i] = txn.(*transaction.Transaction).Hash
	}
	logging.Logger.Debug("delete txns", zap.Any("txns", txnHashes))
	if err := transactionMetadataProvider.GetStore().MultiDelete(ctx, transactionMetadataProvider, txns); err != nil {
		return err
	}

	transaction.RemoveInvalidTxnsFromCache(txnHashes)
	return nil
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
		return nil
	}

	if b.MagicBlock == nil {
		return nil
	}

	mb := b.MagicBlock

	// persist the MB whenever see it. Miners could restart with lfb with this mb_number
	if err = StoreMagicBlock(ctx, mb); err != nil {
		return common.NewErrorf("view_change", "saving MB data: %v", err)
	}

	if !mb.Miners.HasNode(node.Self.Underlying().GetKey()) {
		logging.Logger.Error("[mvc] view change, magic miners does not have self node")
		return // node leaves BC, don't do anything here
	}

	if mc.isSyncingBlocks() {
		return nil
	}

	dkgSum, err := LoadDKGSummary(ctx, strconv.FormatInt(mb.MagicBlockNumber, 10))
	if err != nil {
		logging.Logger.Error("[mvc] view change failed to load dkg summary",
			zap.Error(err),
			zap.Int64("mb number", mb.MagicBlockNumber))
		return nil
	}

	dkgSum.IsFinalized = true
	if err := StoreDKGSummary(ctx, dkgSum); err != nil {
		logging.Logger.Error("[mvc] view change failed to update dkg summary",
			zap.Error(err),
			zap.Int64("mb number", mb.MagicBlockNumber))
		return err
	}

	if err := SetDKG(ctx, mb, dkgSum); err != nil {
		logging.Logger.Error("[mvc] view change set dkg failed",
			zap.Int64("mb number", mb.MagicBlockNumber),
			zap.Int64("mb sr", mb.StartingRound),
			zap.Error(err))
		return err
	}

	// Update the latest finalized magic block - this is critical for the chain to
	// recognize the new magic block and properly validate blocks after view change
	mc.SetLatestFinalizedMagicBlock(b)
	logging.Logger.Info("[mvc] view change - set latest finalized magic block",
		zap.Int64("mb number", mb.MagicBlockNumber),
		zap.Int64("mb sr", mb.StartingRound),
		zap.String("mb hash", mb.Hash))

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

func (mc *Chain) RejectNotarizedBlock(_ string) bool {
	return false
}
