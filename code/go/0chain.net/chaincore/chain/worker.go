package chain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"go.uber.org/zap"
)

var UpdateNodes chan int64

func init() {
	UpdateNodes = make(chan int64, 10)
}

/*SetupWorkers - setup a blockworker for a chain */
func (c *Chain) SetupWorkers(ctx context.Context) {
	go c.StatusMonitor(ctx)
	go c.PruneClientStateWorker(ctx)
	go c.SyncLFBStateWorker(ctx)
	go c.blockFetcher.StartBlockFetchWorker(ctx, c)
	go c.StartLFBTicketWorker(ctx, c.GetLatestFinalizedBlock())
	go node.Self.Underlying().MemoryUsage()
}

// StatusMonitor monitors and updates the node connection status on current magic block
func (c *Chain) StatusMonitor(ctx context.Context) {
	mb := c.GetCurrentMagicBlock()
	newMagicBlockCheckTk := time.NewTicker(5 * time.Second)
	cancel := startStatusMonitor(mb, ctx)

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case newRound := <-UpdateNodes:
			newMB := c.GetMagicBlockNoOffset(newRound)
			if newMB == mb {
				continue
			}

			N2n.Debug("Got nodes update",
				zap.Int64("monitoring round", mb.StartingRound),
				zap.Int64("new round", newRound),
				zap.Int64("mb starting round", newMB.StartingRound))

			if newMB.StartingRound < mb.StartingRound {
				continue
			}

			N2n.Info("Restart status monitor - update nodes",
				zap.Int64("update round", newRound),
				zap.Int64("mb starting round", newMB.StartingRound))
			cancel()
			mb = newMB
			cancel = startStatusMonitor(newMB, ctx)
		case <-newMagicBlockCheckTk.C:
			cmb := c.GetCurrentMagicBlock()
			// current magic block may be kicked back, restart if changed.
			N2n.Debug("new mb status monitor ticker",
				zap.Int64("current mb starting round", cmb.StartingRound),
				zap.Int64("monitoring round", mb.StartingRound))

			if cmb == mb {
				continue
			}

			N2n.Info("Restart status monitor - new mb detected",
				zap.Int64("starting round", cmb.StartingRound),
				zap.Int64("previous starting round", mb.StartingRound))
			cancel()
			mb = cmb
			cancel = startStatusMonitor(cmb, ctx)
		}
	}
}

func startStatusMonitor(mb *block.MagicBlock, ctx context.Context) func() {
	var smctx context.Context
	smctx, cancelCtx := context.WithCancel(ctx)
	waitMC := make(chan struct{})
	waitSC := make(chan struct{})
	go mb.Miners.StatusMonitor(smctx, mb.StartingRound, waitMC)
	go mb.Sharders.StatusMonitor(smctx, mb.StartingRound, waitSC)
	return func() {
		N2n.Debug("[monitor] cancel status monitor", zap.Int64("starting round", mb.StartingRound))
		cancelCtx()
		select {
		case <-waitMC:
		default:
		}

		select {
		case <-waitSC:
		default:
		}
	}
}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) FinalizeRoundWorker(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-c.finalizedRoundsChannel:
			func() {
				// TODO: make the timeout configurable
				cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()
				doneC := make(chan struct{})
				go func() {
					defer close(doneC)
					c.finalizeRound(cctx, r)
					c.UpdateRoundInfo(r)
				}()

				select {
				case <-cctx.Done():
					Logger.Warn("FinalizeRoundWorker finalize round timeout",
						zap.Int64("round", r.GetRoundNumber()))
				case <-doneC:
				}
			}()
		}
	}
}

// MagicBlockBrief represents base info of magic block
type MagicBlockBrief struct {
	MagicBlockNumber int64
	Round            int64
	StartingRound    int64
	MagicBlockHash   string
	MinersN2NURLs    []string
	ShardersN2NURLs  []string
}

// GetLatestFinalizedMagicBlockBrief returns a brief info of the MagicBlock
// to avoid the heavy copy action of the whole block
func (c *Chain) GetLatestFinalizedMagicBlockBrief() *MagicBlockBrief {
	c.lfmbMutex.RLock()
	defer c.lfmbMutex.RUnlock()
	return getMagicBlockBrief(c.latestFinalizedMagicBlock)
}

func (c *Chain) repairChain(ctx context.Context, newMB *block.Block,
	saveFunc MagicBlockSaveFunc) (err error) {

	lfmb := c.GetLatestFinalizedMagicBlockBrief()

	if newMB.MagicBlockNumber <= lfmb.MagicBlockNumber {
		return common.NewError("repair_mb_chain", "already have such MB")
	}

	if newMB.MagicBlockNumber == lfmb.MagicBlockNumber+1 {
		if newMB.PreviousMagicBlockHash != lfmb.MagicBlockHash {
			return common.NewError("repair_mb_chain", "invalid prev-MB ref.")
		}
		return // it's just next MB
	}

	// here the newBM is not next but newer

	Logger.Info("repair_mb_chain: repair from-to mb_number",
		zap.Int64("from", lfmb.MagicBlockNumber),
		zap.Int64("to", newMB.MagicBlockNumber))

	// until the end of the days
	if err = c.VerifyChainHistoryAndRepair(ctx, newMB, saveFunc); err != nil {
		Logger.Error("repair_mb_chain", zap.Error(err))
		return common.NewErrorf("repair_mb_chain", err.Error())
	}

	// the VerifyChainHistoryAndRepair doesn't save the newMB
	// finalizeRound will do it next step

	return // ok
}

// FinalizedBlockWorker - a worker that processes finalized blocks.
func (c *Chain) FinalizedBlockWorker(ctx context.Context, bsh BlockStateHandler) {
	for {
		select {
		case <-ctx.Done():
			return

		case fb := <-c.finalizedBlocksChannel:
			func() {
				// TODO: make the timeout configurable
				cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				t := time.Now()
				doneC := make(chan struct{})
				go func() {
					defer close(doneC)
					c.finalizeBlockProcess(cctx, fb, bsh)
				}()

				select {
				case <-doneC:
					Logger.Debug("finalize block process duration", zap.Any("duration", time.Since(t)))
				case <-cctx.Done():
					Logger.Warn("finalize block process context done",
						zap.Error(cctx.Err()))
				}
			}()
		}
	}
}

func (c *Chain) finalizeBlockProcess(ctx context.Context, fb *block.Block, bsh BlockStateHandler) {
	lfb := c.GetLatestFinalizedBlock()
	if fb.Round < lfb.Round-5 {
		Logger.Error("slow finalized block processing",
			zap.Int64("lfb", lfb.Round), zap.Int64("fb", fb.Round))
	}
	Logger.Debug("Get finalized block from channel", zap.Int64("round", fb.Round))
	// TODO/TOTHINK: move the repair chain outside the finalized worker?

	// make sure we have valid verified MB chain if the block contains
	// a magic block; we already have verified and valid MB chain at this
	// moment, let's keep it updated and verified too

	if fb.MagicBlock != nil && node.Self.Type == node.NodeTypeSharder {
		var err = c.repairChain(ctx, fb, bsh.SaveMagicBlock())
		if err != nil {
			Logger.Error("repairing MB chain", zap.Error(err))
			return
		}
	}

	// finalize
	if !fb.IsStateComputed() {
		Logger.Debug("finalize block state not computed",
			zap.Int64("round", fb.Round))
		err := c.ComputeOrSyncState(ctx, fb)
		if err != nil {
			Logger.Error("save changes - save state not successful",
				zap.Int64("round", fb.Round),
				zap.String("hash", fb.Hash),
				zap.Int8("state", fb.GetBlockState()),
				zap.Error(err))
			if state.Debug() {
				Logger.DPanic("save changes - state not successful")
			}
		}
	} else {
		Logger.Debug("finalize block state computed",
			zap.Int64("round", fb.Round),
			zap.Any("state", fb.GetStateStatus()))
	}

	switch fb.GetStateStatus() {
	case block.StateSynched, block.StateSuccessful:
	default:
		Logger.Error("state_save_without_success, state can't be saved without successful computation",
			zap.Int64("round", fb.Round))
		return
	}

	// Fetch block state changes and apply them would reduce the blocks finalize speed
	if fb.ClientState == nil {
		Logger.Error("Finalize block - client state is null, get state changes from network",
			zap.Int64("round", fb.Round),
			zap.String("hash", fb.Hash))
		if err := c.GetBlockStateChange(fb); err != nil {
			Logger.Error("Finalize block - get block state changes failed",
				zap.Error(err),
				zap.Int64("round", fb.Round),
				zap.String("block hash", fb.Hash))
			return
		}
	}

	c.finalizeBlock(ctx, fb, bsh)
}

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	tick := time.Duration(c.PruneStateBelowCount) * time.Second
	timer := time.NewTimer(time.Second)
	pruning := false
	Logger.Debug("PruneClientStateWorker start")
	defer func() {
		Logger.Debug("PruneClientStateWorker stopped, we should not see this...")
	}()

	for true {
		select {
		case <-timer.C:
			Logger.Debug("Do prune client state worker")
			if pruning {
				Logger.Info("pruning still going on")
				continue
			}
			pruning = true
			c.pruneClientState(ctx)
			pruning = false
			if c.pruneStats == nil || c.pruneStats.MissingNodes > 0 {
				timer = time.NewTimer(time.Second)
			} else {
				timer = time.NewTimer(tick)
			}
		}
	}
}

// SyncLFBStateWorker is a worker for syncing state of latest finalized round block.
// The worker would not sync state for every LFB as it will cause performance issue,
// only when it detects BC stuck will the synch process start.
func (c *Chain) SyncLFBStateWorker(ctx context.Context) {
	Logger.Debug("SyncLFBStateWorker start")
	defer func() {
		Logger.Debug("SyncLFBStateWorker stopped")
	}()

	lfb := c.GetLatestFinalizedBlock()

	// lastRound records the last latest finalized round info, which will be
	// updated once a new LFB is found. If its timestamp is not updated for specific
	// time duration (100s currently), we can say the BC is stuck, and the process for
	// syncing state will be triggered.
	var lastRound = struct {
		round     int64
		stateHash util.Key
		tm        time.Time
	}{
		round:     lfb.Round,
		stateHash: lfb.ClientStateHash,
		tm:        time.Now(),
	}

	// context and cancel function will be used to cancel a running state syncing process when
	// the BC starts to move again.
	var cctx context.Context
	var cancel context.CancelFunc

	// ticker to check if the BC is stuck
	tk := time.NewTicker(c.bcStuckCheckInterval)
	var isSynching bool
	synchingStopC := make(chan struct{})

	for {
		select {
		case bs := <-c.syncLFBStateC:
			// got a new finalized block summary
			if bs.Round > lastRound.round && lastRound.round > 0 {
				Logger.Debug("BC is moving",
					zap.Int64("current_lfb_round", bs.Round),
					zap.Int64("last_round", lastRound.round))
				// call cancel to stop state syncing process in case it was started
				if cancel != nil && isSynching {
					cancel()
					cancel = nil
				}

				// update to latest finalized round
				lastRound.round = bs.Round
				lastRound.stateHash = bs.ClientStateHash
				lastRound.tm = time.Now()
				continue
			}
		case <-tk.C:
			// last round could be 0 when miners or sharders start
			lfb := c.GetLatestFinalizedBlock()
			if lastRound.round == 0 {
				lastRound.round = lfb.Round
				lastRound.stateHash = lfb.ClientStateHash
				lastRound.tm = time.Now()
				continue
			}

			// time since the last finalized round arrived
			ts := time.Since(lastRound.tm)
			if ts <= c.bcStuckTimeThreshold {
				// reset sync state and continue as the BC is not stuck
				isSynching = false
				continue
			}

			// continue if state is syncing
			if isSynching {
				continue
			}

			Logger.Debug("BC may get stuck",
				zap.Int64("lastRound", lastRound.round),
				zap.String("state_hash", util.ToHex(lastRound.stateHash)),
				zap.Any("stuck time", ts))

			cctx, cancel = context.WithCancel(ctx)
			isSynching = true
			go func() {
				defer func() {
					synchingStopC <- struct{}{}
				}()
				if lfb == nil {
					return
				}
				mpt, err := c.syncRoundStateToStateDB(cctx, lfb.Round, lfb.ClientStateHash)
				if err != nil {
					Logger.Error("sync round state failed", zap.Error(err))
					return
				}
				if err := c.UpdateLatestFinalizedBlockState(mpt); err != nil {
					Logger.Error("update latest finalized block state failed", zap.Error(err))
				}
			}()
		case <-synchingStopC:
			isSynching = false
		case <-ctx.Done():
			Logger.Info("Context done, stop SyncLFBStateWorker")
			cancel()
			return
		}
	}
}

func (c *Chain) syncRoundStateToStateDB(ctx context.Context, round int64, rootStateHash util.Key) (util.MerklePatriciaTrieI, error) {
	Logger.Info("Sync round state from network...", zap.Int64("round", round))
	mpt := util.NewMerklePatriciaTrie(c.GetStateDB(), util.Sequence(round))
	mpt.SetRoot(rootStateHash)

	Logger.Info("Finding missing nodes")
	cctx, cancel := context.WithTimeout(ctx, c.syncStateTimeout)
	defer cancel()

	_, keys, err := mpt.FindMissingNodes(cctx)
	if err != nil {
		switch err {
		case context.Canceled:
			return nil, common.NewError("sync_round_state_abort", "context is canceled, suppose the BC is moving")
		case context.DeadlineExceeded:
			return nil, common.NewError("sync round state abort", "context timed out for checking missing nodes")
		default:
			return nil, common.NewError("sync round state abort",
				fmt.Sprintf("failed to get missing nodes, round: %d, client state hash: %s, err: %v",
					round, util.ToHex(rootStateHash), err))
		}
	}

	if len(keys) == 0 {
		Logger.Debug("Found no missing node",
			zap.Int64("round", round),
			zap.String("state hash", util.ToHex(rootStateHash)))
		return mpt, nil
	}

	Logger.Info("Sync round state, found missing nodes",
		zap.Int64("round", round),
		zap.Int("missing_node_num", len(keys)))

	if err := c.UpdateStateFromNetwork(ctx, mpt, keys); err != nil {
		return nil, common.NewError("update state from network failed",
			fmt.Sprintf("round: %d, client state hash: %s, err: %v",
				round, util.ToHex(rootStateHash), err))
	}

	return mpt, nil
}

type MagicBlockSaveFunc func(context.Context, *block.Block) error

// VerifyChainHistoryAndRepairOn repairs and verifies magic blocks chain using given
// current MagicBlock to request other nodes.
func (c *Chain) VerifyChainHistoryAndRepairOn(ctx context.Context,
	latestMagicBlock *block.Block,
	cmb *block.MagicBlock,
	saveHandler MagicBlockSaveFunc) (err error) {

	var (
		currentLFMB = c.GetLatestFinalizedMagicBlock()
		sharders    = cmb.Sharders.N2NURLs()
		magicBlock  *block.Block
	)

	// until we have got all MB from our from store to latest given
	for currentLFMB.Hash != latestMagicBlock.Hash {
		if currentLFMB.MagicBlockNumber > latestMagicBlock.MagicBlockNumber {
			err = errors.New("verify chain history failed, latest magic block ")
			Logger.Debug("current lfmb number is greater than new lfmb number",
				zap.Int64("current_lfmb_number", currentLFMB.MagicBlockNumber),
				zap.Int64("new lfmb_number", latestMagicBlock.MagicBlockNumber),
				zap.Int64("current_lfmb_round", currentLFMB.Round),
				zap.Int64("new lfmb_round", latestMagicBlock.Round))
			return
		}

		if currentLFMB.MagicBlockNumber == latestMagicBlock.MagicBlockNumber {
			err = errors.New("verify chain history failed, latest magic block does not match")
			Logger.Error("verify_chain_history failed",
				zap.Error(err),
				zap.String("current_lfmb_hash", currentLFMB.Hash),
				zap.String("latest_mb_hash", latestMagicBlock.Hash),
				zap.Int64("magic block number", currentLFMB.MagicBlockNumber))
			return
		}

		requestMBNum := currentLFMB.MagicBlockNumber + 1
		Logger.Debug("verify_chain_history", zap.Int64("get_mb_number", requestMBNum))

		// magicBlock, err = httpclientutil.GetMagicBlockCall(sharders, requestMBNum, 1)
		magicBlock, err = httpclientutil.FetchMagicBlockFromSharders(ctx, sharders, requestMBNum)

		if err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to get %d: %v", requestMBNum, err))
		}

		if !currentLFMB.VerifyMinersSignatures(magicBlock) {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to verify magic block %d miners signatures", requestMBNum))
		}

		Logger.Info("verify chain history",
			zap.Any("mb_sr", magicBlock.StartingRound),
			zap.Any("mb_hash", magicBlock.Hash))

		if err = c.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to update magic block %d: %v", requestMBNum, err))
		}

		c.SetLatestFinalizedMagicBlock(magicBlock)
		currentLFMB = magicBlock

		if saveHandler != nil {
			if err = saveHandler(ctx, magicBlock); err != nil {
				return common.NewError("get_lfmb_from_sharders",
					fmt.Sprintf("failed to save updated magic block %d: %v",
						currentLFMB.MagicBlockNumber, err))
			}
		}

	}

	return
}

// VerifyChainHistoryAndRepair repairs and verifies magic blocks chain. It uses
// GetCurrnetMagicBlock to get sharders to request data from.
func (c *Chain) VerifyChainHistoryAndRepair(ctx context.Context,
	latestMagicBlock *block.Block, saveHandler MagicBlockSaveFunc) (err error) {

	return c.VerifyChainHistoryAndRepairOn(ctx, latestMagicBlock,
		c.GetCurrentMagicBlock(), saveHandler)
}

// PruneStorageWorker pruning storage
func (c *Chain) PruneStorageWorker(ctx context.Context, d time.Duration,
	getCountRoundStorage func(storage round.RoundStorage) int,
	storage ...round.RoundStorage) {
	ticker := time.NewTicker(d)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.PruneRoundStorage(ctx, getCountRoundStorage, storage...)
		}
	}
}

// MagicBlockSaver represents a node with ability to save a received and
// verified magic block.
type MagicBlockSaver interface {
	SaveMagicBlock() MagicBlockSaveFunc // get the saving function
}

// UpdateLatesMagicBlockFromShardersOn pulls latest finalized magic block
// from sharders and verifies magic blocks chain. The method blocks
// execution flow (it's synchronous). It uses given MagicBlock to get list
// of sharders to request.
func (sc *Chain) UpdateLatesMagicBlockFromShardersOn(ctx context.Context,
	mb *block.MagicBlock) (err error) {

	block := sc.GetLatestFinalizedMagicBlockFromShardersOn(ctx, mb)
	if block == nil {
		Logger.Warn("no new finalized magic block from sharders given",
			zap.Strings("URLs", mb.Sharders.N2NURLs()))
		return nil
	}

	cmb := sc.GetCurrentMagicBlock()

	Logger.Info("get current magic block from sharders",
		zap.Any("number", block.MagicBlockNumber),
		zap.Any("sr", block.StartingRound),
		zap.Any("hash", block.Hash))

	if block.MagicBlock.StartingRound <= cmb.StartingRound {
		if block.MagicBlock.StartingRound == cmb.StartingRound && block.MagicBlock.Hash == cmb.Hash {
			block.MagicBlock = cmb
			sc.SetLatestFinalizedMagicBlock(block)
			Logger.Debug(
				"updated lfmb to add lfmb's parent block to magicBlockStartRounds cache",
				zap.Any("block hash", block.Hash),
				zap.Any("block round", block.Round),
				zap.Any("lfmb starting round", block.StartingRound),
			)
		}
		return nil // earlier than the current one
	}

	var saveMagicBlock MagicBlockSaveFunc
	if sc.magicBlockSaver != nil {
		saveMagicBlock = sc.magicBlockSaver.SaveMagicBlock()
	}

	err = sc.VerifyChainHistoryAndRepair(ctx, block, saveMagicBlock)
	if err != nil {
		return fmt.Errorf("failed to verify chain history: %v", err.Error())
	}

	if err = sc.UpdateMagicBlock(block.MagicBlock); err != nil {
		return fmt.Errorf("failed to update magic block: %v", err.Error())
	}
	sc.SetLatestFinalizedMagicBlock(block)

	return // ok, updated
}

// UpdateLatesMagicBlockFromSharders pulls latest finalized magic block
// from sharders and verifies magic blocks chain. The method blocks
// execution flow (it's synchronous).
func (sc *Chain) UpdateLatesMagicBlockFromSharders(ctx context.Context) (
	err error) {
	return sc.UpdateLatesMagicBlockFromShardersOn(ctx, sc.GetLatestMagicBlock())
}

// UpdateMagicBlockWorker updates latest finalized magic block from active
// sharders periodically.
func (c *Chain) UpdateMagicBlockWorker(ctx context.Context) {

	var (
		tick = time.NewTicker(5 * time.Second)

		tickq = tick.C
		doneq = ctx.Done()

		err error
	)

	defer tick.Stop()

	for {
		select {
		case <-doneq:
			return
		case <-tickq:
		}

		if err = c.UpdateLatesMagicBlockFromSharders(ctx); err != nil {
			Logger.Error("update_mb_worker", zap.Error(err))
		}
	}

}
