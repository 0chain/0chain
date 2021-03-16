package chain

import (
	"context"
	"fmt"
	"sort"
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

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) StatusMonitor(ctx context.Context) {
	smctx, cancel := context.WithCancel(ctx)
	mb := c.GetCurrentMagicBlock()
	go mb.Miners.StatusMonitor(smctx)
	go mb.Sharders.StatusMonitor(smctx)
	for true {
		select {
		case <-ctx.Done():
			return
		case nRound := <-UpdateNodes:
			cancel()
			Logger.Info("the status monitor is dead, long live the status monitor",
				zap.Any("miners", mb.Miners), zap.Any("sharders", mb.Sharders))
			smctx, cancel = context.WithCancel(ctx)
			mb := c.GetMagicBlock(nRound)
			go mb.Miners.StatusMonitor(smctx)
			go mb.Sharders.StatusMonitor(smctx)
		}
	}
}

/*FinalizeRoundWorker - a worker that handles the finalized blocks */
func (c *Chain) FinalizeRoundWorker(ctx context.Context, bsh BlockStateHandler) {
	for {
		select {
		case <-ctx.Done():
			return
		case r := <-c.finalizedRoundsChannel:
			c.finalizeRound(ctx, r, bsh)
			c.UpdateRoundInfo(r)
		}
	}
}

func (c *Chain) repairChain(ctx context.Context, newMB *block.Block,
	saveFunc MagicBlockSaveFunc) (err error) {

	var latest = c.GetLatestFinalizedMagicBlock()

	if newMB.MagicBlockNumber <= latest.MagicBlockNumber {
		return common.NewError("repair_mb_chain", "already have such MB")
	}

	if newMB.MagicBlockNumber == latest.MagicBlockNumber+1 {
		if newMB.PreviousMagicBlockHash != latest.MagicBlock.Hash {
			return common.NewError("repair_mb_chain", "invalid prev-MB ref.")
		}
		return // it's just next MB
	}

	// here the newBM is not next but newer

	Logger.Info("repair_mb_chain: repair from-to mb_number",
		zap.Int64("from", latest.MagicBlockNumber),
		zap.Int64("to", newMB.MagicBlockNumber))

	// until the end of the days
	if err = c.VerifyChainHistory(ctx, newMB, saveFunc); err != nil {
		Logger.Error("repair_mb_chain", zap.Error(err))
		return common.NewErrorf("repair_mb_chain", err.Error())
	}

	// the VerifyChainHistory doesn't save the newMB
	// finalizeRound will do it next step

	return // ok
}

// FinalizedBlockWorker - a worker that processes finalized blocks.
func (c *Chain) FinalizedBlockWorker(ctx context.Context,
	bsh BlockStateHandler) {

	for {
		select {
		case <-ctx.Done():
			return

		case fb := <-c.finalizedBlocksChannel:
			lfb := c.GetLatestFinalizedBlock()
			if fb.Round < lfb.Round-5 {
				Logger.Error("slow finalized block processing",
					zap.Int64("lfb", lfb.Round), zap.Int64("fb", fb.Round))
			}

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
			}

			switch fb.GetStateStatus() {
			case block.StateSynched, block.StateSuccessful:
			default:
				Logger.Error("state_save_without_success, state can't be saved without successful computation")
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
	}
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
			if lastRound.round == 0 {
				lfb := c.GetLatestFinalizedBlock()
				lastRound.round = lfb.Round
				lastRound.stateHash = lfb.ClientStateHash
				lastRound.tm = time.Now()
				continue
			}

			// time since the last finalized round arrived
			ts := time.Since(lastRound.tm)
			if ts <= c.bcStuckTimeThreshold {
				// reset synching state and continue as the BC is not stuck
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

			r := lastRound.round
			stateHash := lastRound.stateHash
			cctx, cancel = context.WithCancel(ctx)

			isSynching = true
			go func() {
				c.syncRoundState(cctx, r, stateHash)
				synchingStopC <- struct{}{}
			}()
		case <-synchingStopC:
			isSynching = false
		case <-ctx.Done():
			Logger.Info("Context done, stop SyncLFBStateWorker")
			return
		}
	}
}

func (c *Chain) syncRoundState(ctx context.Context, round int64, stateRootHash util.Key) {
	Logger.Info("Sync round state from network...")
	mpt := util.NewMerklePatriciaTrie(c.stateDB, util.Sequence(round))
	mpt.SetRoot(stateRootHash)

	Logger.Info("Finding missing nodes")
	cctx, cancel := context.WithTimeout(ctx, c.syncStateTimeout)
	defer cancel()

	_, keys, err := mpt.FindMissingNodes(cctx)
	if err != nil {
		switch err {
		case context.Canceled:
			Logger.Error("Sync round state abort, context is canceled, suppose the BC is moving")
			return
		case context.DeadlineExceeded:
			Logger.Error("Sync round state abort, context timed out for checking missing nodes")
			return
		default:
			Logger.Error("Sync round state abort, failed to get missing nodes",
				zap.Int64("round", round),
				zap.String("client state hash", util.ToHex(stateRootHash)),
				zap.Error(err))
			return
		}
	}

	if len(keys) == 0 {
		Logger.Debug("Found no missing node",
			zap.Int64("round", round),
			zap.String("state hash", util.ToHex(stateRootHash)))
		return
	}

	Logger.Info("Sync round state, found missing nodes",
		zap.Int64("round", round),
		zap.Int("missing_node_num", len(keys)))

	c.GetStateNodesFromSharders(ctx, keys)
}

type MagicBlockSaveFunc func(context.Context, *block.Block) error

// VerifyChainHistoryOn repairs and verifies magic blocks chain using given
// current MagicBlock to request other nodes.
func (c *Chain) VerifyChainHistoryOn(ctx context.Context,
	latestMagicBlock *block.Block, cmb *block.MagicBlock,
	saveHandler MagicBlockSaveFunc) (err error) {

	var (
		currentMagicBlock = c.GetLatestFinalizedMagicBlock()
		sharders          = cmb.Sharders.N2NURLs()
		magicBlock        *block.Block
	)

	// until we have got all MB from our from store to latest given
	for currentMagicBlock.Hash != latestMagicBlock.Hash {
		Logger.Debug("verify_chain_history",
			zap.Int64("get_mb_number", currentMagicBlock.MagicBlockNumber+1))

		magicBlock, err = httpclientutil.GetMagicBlockCall(sharders,
			currentMagicBlock.MagicBlockNumber+1, 1)

		if err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to get %d: %v",
					currentMagicBlock.MagicBlockNumber+1, err))
		}

		if !currentMagicBlock.VerifyMinersSignatures(magicBlock) {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to verify magic block %d miners signatures",
					currentMagicBlock.MagicBlockNumber+1))
		}

		Logger.Info("verify chain history",
			zap.Any("mb_sr", magicBlock.StartingRound),
			zap.Any("mb_hash", magicBlock.Hash))

		if err = c.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to update magic block %d: %v",
					currentMagicBlock.MagicBlockNumber+1, err))
		} else {
			c.UpdateNodesFromMagicBlock(magicBlock.MagicBlock)
		}

		c.SetLatestFinalizedMagicBlock(magicBlock)
		currentMagicBlock = magicBlock

		if saveHandler != nil {
			if err = saveHandler(ctx, magicBlock); err != nil {
				return common.NewError("get_lfmb_from_sharders",
					fmt.Sprintf("failed to save updated magic block %d: %v",
						currentMagicBlock.MagicBlockNumber, err))
			}
		}

	}

	return
}

// VerifyChainHistory repairs and verifies magic blocks chain. It uses
// GetCurrnetMagicBlock to get sharders to request data from.
func (c *Chain) VerifyChainHistory(ctx context.Context,
	latestMagicBlock *block.Block, saveHandler MagicBlockSaveFunc) (err error) {

	return c.VerifyChainHistoryOn(ctx, latestMagicBlock,
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

	var mbs = sc.GetLatestFinalizedMagicBlockFromShardersOn(ctx, mb)
	if len(mbs) == 0 {
		return fmt.Errorf("no finalized magic block from sharders (%s) given",
			mb.Sharders.N2NURLs())
	}

	if len(mbs) > 1 {
		sort.Slice(mbs, func(i, j int) bool {
			return mbs[i].StartingRound > mbs[j].StartingRound
		})
	}

	var (
		magicBlock = mbs[0]
		cmb        = sc.GetCurrentMagicBlock()
	)

	Logger.Info("get current magic block from sharders",
		zap.Any("number", magicBlock.MagicBlockNumber),
		zap.Any("sr", magicBlock.StartingRound),
		zap.Any("hash", magicBlock.Hash))

	if magicBlock.StartingRound <= cmb.StartingRound {
		return nil // earlier than the current one
	}

	var saveMagicBlock MagicBlockSaveFunc
	if sc.magicBlockSaver != nil {
		saveMagicBlock = sc.magicBlockSaver.SaveMagicBlock()
	}

	err = sc.VerifyChainHistory(ctx, magicBlock, saveMagicBlock)
	if err != nil {
		return fmt.Errorf("failed to verify chain history: %v", err.Error())
	}

	if err = sc.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
		return fmt.Errorf("failed to update magic block: %v", err.Error())
	}

	sc.UpdateNodesFromMagicBlock(magicBlock.MagicBlock)
	sc.SetLatestFinalizedMagicBlock(magicBlock)

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
