package chain

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
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
	go c.blockFetcher.StartBlockFetchWorker(ctx, c)
	go c.StartLFBTicketWorker(ctx, c.GetLatestFinalizedBlock())
	go node.Self.Underlying().MemoryUsage()
}

// StatusMonitor monitors and updates the node connection status on current magic block
func (c *Chain) StatusMonitor(ctx context.Context) {
	mb := c.getLatestFinalizedMagicBlock(ctx)
	newMagicBlockCheckTk := time.NewTicker(5 * time.Second)
	var cancel func()
	if mb != nil {
		cancel = startStatusMonitor(mb, ctx)
	}

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		case newStartingRound := <-UpdateNodes:
			newMB := c.GetMagicBlockNoOffset(newStartingRound)
			if newMB == nil {
				continue
			}

			if newMB == mb {
				continue
			}

			if mb == nil {
				mb = newMB
				cancel = startStatusMonitor(newMB, ctx)
				continue
			}

			if newMB.StartingRound < mb.StartingRound {
				continue
			}

			Logger.Debug("[monitor] got new magic block, update nodes",
				zap.Int64("monitoring round", mb.StartingRound),
				zap.Int64("new mb starting round", newMB.StartingRound))

			cancel()
			mb = newMB
			cancel = startStatusMonitor(newMB, ctx)
		case <-newMagicBlockCheckTk.C:
			cmb := c.getLatestFinalizedMagicBlock(ctx)
			if cmb == nil {
				continue
			}
			if cmb == mb {
				continue
			}

			Logger.Info("[monitor] restart status monitor - new mb detected",
				zap.Int64("monitoring starting round", mb.StartingRound),
				zap.Int64("new mb starting round", cmb.StartingRound))
			cancel()
			mb = cmb
			cancel = startStatusMonitor(cmb, ctx)
		}
	}
}

func startStatusMonitor(mb *block.MagicBlock, ctx context.Context) func() {
	Logger.Info("[monitor] start status monitor - update nodes",
		zap.Int64("mb starting round", mb.StartingRound))
	var smctx context.Context
	smctx, cancelCtx := context.WithCancel(ctx)
	waitMC := make(chan struct{})
	waitSC := make(chan struct{})
	go mb.Miners.StatusMonitor(smctx, mb.StartingRound, waitMC)
	go mb.Sharders.StatusMonitor(smctx, mb.StartingRound, waitSC)
	return func() {
		Logger.Info("[monitor] cancel status monitor", zap.Int64("starting round", mb.StartingRound))
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
	var (
		finalizingRound    int64
		cancel             func()
		finalizingC        = make(chan round.RoundI, 2*config.GetLFBTicketAhead()+1)
		getFinalizingRound = func() int64 {
			return atomic.LoadInt64(&finalizingRound)
		}
		setFinalizingRound = func(r int64) {
			atomic.StoreInt64(&finalizingRound, r)
		}
	)

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case r := <-finalizingC:
				func() {
					setFinalizingRound(r.GetRoundNumber())
					// TODO: make the timeout configurable
					var cctx context.Context
					cctx, cancel = context.WithTimeout(ctx, time.Minute)
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
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case r := <-c.finalizedRoundsChannel:
			rn := r.GetRoundNumber()
			fr := getFinalizingRound()
			if fr > 0 && rn-fr > int64(2*config.GetLFBTicketAhead()) {
				// drain out finalizing round channel
				lc := len(finalizingC)
				for i := 0; i < lc; i++ {
					<-finalizingC
				}

				// cancel and force move the finalizing round to current round
				if cancel != nil {
					cancel()
				}

				Logger.Debug("FinalizeRoundWorker - finalizing round slow, do fast moving",
					zap.Int64("to round", rn),
					zap.Int64("finalizing round", fr))
			}

			finalizingC <- r
			continue
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
	return getMagicBlockBrief(c.GetLatestFinalizedMagicBlock(context.Background()))
}

func (c *Chain) repairChain(ctx context.Context, newMB *block.Block,
	saveFunc MagicBlockSaveFunc) (err error) {

	lfmb := c.GetLatestFinalizedMagicBlockBrief()
	if lfmb == nil {
		return common.NewError("repair_mb_chain", "can't get lfmb")
	}

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

		case fbr := <-c.finalizedBlocksChannel:
			func() {
				// TODO: make the timeout configurable
				cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
				defer cancel()

				errC := make(chan error, 1)
				go func() {
					errC <- c.finalizeBlockProcess(cctx, fbr.block, bsh)
				}()

				select {
				case err := <-errC:
					fbr.resultC <- err
				case <-cctx.Done():
					Logger.Warn("finalize block process context done",
						zap.Error(cctx.Err()))
					fbr.resultC <- cctx.Err()
				}
			}()
		}
	}
}

func (c *Chain) finalizeBlockProcess(ctx context.Context, fb *block.Block, bsh BlockStateHandler) error {
	lfb := c.GetLatestFinalizedBlock()
	if fb.Round < lfb.Round-5 {
		Logger.Warn("finalize block - slow finalized block processing",
			zap.Int64("lfb", lfb.Round), zap.Int64("fb", fb.Round))
	}

	if lfb.Round == fb.Round && lfb.Hash == fb.Hash {
		Logger.Info("finalize block - already finalized",
			zap.Int64("round", fb.Round),
			zap.String("block", fb.Hash))
		return nil
	}

	Logger.Debug("start to finalize block",
		zap.Int64("round", fb.Round),
		zap.String("block", fb.Hash),
		zap.String("prev block", fb.PrevHash))

	isSharder := node.Self.IsSharder()

	if !fb.IsStateComputed() {
		if fb.PrevBlock == nil {
			pb := c.GetLocalPreviousBlock(ctx, fb)
			if isSharder {
				if pb == nil || !pb.IsStateComputed() {
					Logger.Error("finalize block - no previous block ready",
						zap.Int64("round", fb.Round),
						zap.String("block", fb.Hash),
						zap.String("prev block", fb.PrevHash),
						zap.Int64("lfb round", lfb.Round),
						zap.String("lfb", lfb.Hash))
					return errors.New("previous block state not computed or synced")
				}
			}

			if pb != nil {
				fb.SetPreviousBlock(pb)
			}
		}

		if isSharder {
			// compute state
			if err := c.ComputeState(ctx, fb); err != nil {
				Logger.Error("finalize block - compute state failed",
					zap.Int64("round", fb.Round),
					zap.Error(err))
				return fmt.Errorf("compute state failed: %v", err)
			}
		} else {
			Logger.Debug("finalize block - state not computed, try to fetch state changes",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash),
				zap.String("prev block", fb.PrevHash))

			if err := c.GetBlockStateChange(fb); err != nil {
				Logger.Error("finalize block failed, compute state failed",
					zap.Int64("round", fb.Round),
					zap.Error(err))
				return fmt.Errorf("sync state changes failed: %v", err)
			}

			Logger.Debug("finalize block - sync state success",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash))
		}
	}

	// TODO/TOTHINK: move the repair chain outside the finalized worker?
	// make sure we have valid verified MB chain if the block contains
	// a magic block; we already have verified and valid MB chain at this
	// moment, let's keep it updated and verified too

	if fb.MagicBlock != nil && node.Self.Type == node.NodeTypeSharder {
		var err = c.repairChain(ctx, fb, bsh.SaveMagicBlock())
		if err != nil {
			Logger.Error("finalize block - repairing MB chain", zap.Error(err))
			return fmt.Errorf("repair chain failed: %v", err)
		}
	}

	if isSharder {
		// get previous finalized block
		pr := c.GetRound(fb.Round - 1)
		if pr == nil {
			Logger.Error("finalize block - previous round not found",
				zap.Int64("round", fb.Round))
			return errors.New("previous round is missing")
		}

		prevBlockHash := pr.GetBlockHash()
		if prevBlockHash == "" {
			Logger.Error("finalize block - previous round not finalized",
				zap.Int64("round", fb.Round))
			return errors.New("previous round not finalized")
		}

		if fb.PrevHash != prevBlockHash {
			Logger.Error("finalize block - could not connect to lfb",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash),
				zap.String("prev block", fb.PrevHash),
				zap.String("finalized previous block", prevBlockHash))
			return errors.New("could not connect to lfb")
		}

	}
	// finalize
	c.finalizeBlock(ctx, fb, bsh)
	return nil
}

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	tick := 30 * time.Second
	timer := time.NewTimer(time.Second)
	Logger.Debug("PruneClientStateWorker start")
	defer func() {
		Logger.Debug("PruneClientStateWorker stopped, we should not see this...")
	}()

	for {
		select {
		case <-timer.C:
			Logger.Debug("Do prune client state worker")
			c.pruneClientState(ctx)
			if c.pruneStats == nil {
				timer = time.NewTimer(time.Second)
			} else {
				timer = time.NewTimer(tick)
			}
		//case <-c.pruneClientStateC:
		//	timer.Reset(0)
		case <-ctx.Done():
			return
		}
	}
}

func (c *Chain) StartPruneClientState() {
	c.pruneClientStateC <- struct{}{}
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

				c.syncRoundStateToStateDB(cctx, lfb.Round, lfb.ClientStateHash)
			}()
		case <-c.syncLFBStateNowC:
			if isSynching {
				continue
			}

			lfb := c.GetLatestFinalizedBlock()
			Logger.Info("Sync LFB immediately", zap.Int64("lfb round", lfb.Round),
				zap.Int64("current round", c.GetCurrentRound()))

			cctx, cancel = context.WithCancel(ctx)
			isSynching = true
			go func() {
				defer func() {
					synchingStopC <- struct{}{}
				}()
				if lfb == nil {
					return
				}

				c.syncRoundStateToStateDB(cctx, lfb.Round, lfb.ClientStateHash)
			}()
		case <-synchingStopC:
			isSynching = false
		case <-ctx.Done():
			Logger.Info("Context done, stop SyncLFBStateWorker")
			if cancel != nil {
				cancel()
			}
			return
		}
	}
}

func (c *Chain) syncRoundStateToStateDB(ctx context.Context, round int64, rootStateHash util.Key) {
	Logger.Info("Sync round state from network...")
	mpt := util.NewMerklePatriciaTrie(c.stateDB, util.Sequence(round), rootStateHash)

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
				zap.String("client state hash", util.ToHex(rootStateHash)),
				zap.Error(err))
			return
		}
	}

	if len(keys) == 0 {
		Logger.Debug("Found no missing node",
			zap.Int64("round", round),
			zap.String("state hash", util.ToHex(rootStateHash)))
		return
	}

	Logger.Info("Sync round state, found missing nodes",
		zap.Int64("round", round),
		zap.Int("missing_node_num", len(keys)))

	c.GetStateNodes(ctx, keys)
}

type MagicBlockSaveFunc func(context.Context, *block.Block) error

// VerifyChainHistoryAndRepairOn repairs and verifies magic blocks chain using given
// current MagicBlock to request other nodes.
func (c *Chain) VerifyChainHistoryAndRepairOn(ctx context.Context,
	latestMagicBlock *block.Block,
	cmb *block.MagicBlock,
	saveHandler MagicBlockSaveFunc) (err error) {

	var (
		sharders   = cmb.Sharders.N2NURLs()
		magicBlock *block.Block
	)
	currentLFMB := c.GetLatestFinalizedMagicBlock(ctx)
	if currentLFMB == nil {
		return errors.New("can't get currentLFMB")
	}

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

		magicBlock, err = httpclientutil.FetchMagicBlockFromSharders(ctx, sharders, requestMBNum,
			func(b *block.Block) bool {
				return currentLFMB.VerifyMinersSignatures(b)
			})
		if err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to get %d: %v", requestMBNum, err))
		}

		Logger.Info("verify chain history",
			zap.Any("mb_sr", magicBlock.StartingRound),
			zap.Any("mb_hash", magicBlock.Hash),
			zap.Int64("mb_num", magicBlock.MagicBlockNumber))

		if err = c.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
			Logger.Error("verify chain history - update magic block failed", zap.Error(err))
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
			c.PruneRoundStorage(getCountRoundStorage, storage...)
		}
	}
}

// MagicBlockSaver represents a node with ability to save a received and
// verified magic block.
type MagicBlockSaver interface {
	SaveMagicBlock() MagicBlockSaveFunc // get the saving function
}

// UpdateLatestMagicBlockFromShardersOn pulls latest finalized magic block
// from sharders and verifies magic blocks chain. The method blocks
// execution flow (it's synchronous). It uses given MagicBlock to get list
// of sharders to request.
func (c *Chain) UpdateLatestMagicBlockFromShardersOn(ctx context.Context,
	mb *block.MagicBlock) (err error) {

	lfmb := c.GetLatestFinalizedMagicBlockFromShardersOn(ctx, mb)
	if lfmb == nil {
		Logger.Warn("no new finalized magic lfmb from sharders given",
			zap.Strings("URLs", mb.Sharders.N2NURLs()))
		return nil
	}

	cmb := c.GetLatestFinalizedMagicBlock(ctx)
	if cmb == nil {
		return errors.New("can't get cmb")
	}
	if lfmb.Hash == cmb.Hash {
		return nil
	}

	Logger.Info("get magic lfmb from sharders",
		zap.Any("number", lfmb.MagicBlockNumber),
		zap.Any("sr", lfmb.StartingRound),
		zap.Any("hash", lfmb.Hash),
		zap.Int64("local lfmb", lfmb.StartingRound))

	if lfmb.MagicBlock.StartingRound <= cmb.StartingRound {
		if lfmb.MagicBlock.StartingRound == cmb.StartingRound && lfmb.MagicBlock.Hash == cmb.Hash {
			lfmb.MagicBlock = cmb.MagicBlock
			c.SetLatestFinalizedMagicBlock(lfmb)
			Logger.Debug(
				"updated lfmb to add lfmb's parent lfmb to magicBlockStartRounds cache",
				zap.Any("lfmb hash", lfmb.Hash),
				zap.Any("lfmb round", lfmb.Round),
				zap.Any("lfmb starting round", lfmb.StartingRound),
			)
		}
		return nil // earlier than the current one
	}

	var saveMagicBlock MagicBlockSaveFunc
	if c.magicBlockSaver != nil {
		saveMagicBlock = c.magicBlockSaver.SaveMagicBlock()
	}

	err = c.VerifyChainHistoryAndRepair(ctx, lfmb, saveMagicBlock)
	if err != nil {
		return fmt.Errorf("failed to verify chain history: %v", err.Error())
	}

	if err = c.UpdateMagicBlock(lfmb.MagicBlock); err != nil {
		return fmt.Errorf("failed to update magic lfmb: %v", err.Error())
	}
	c.SetLatestFinalizedMagicBlock(lfmb)

	return // ok, updated
}

// UpdateLatestMagicBlockFromSharders pulls latest finalized magic block
// from sharders and verifies magic blocks chain. The method blocks
// execution flow (it's synchronous).
func (c *Chain) UpdateLatestMagicBlockFromSharders(ctx context.Context) (
	err error) {
	return c.UpdateLatestMagicBlockFromShardersOn(ctx, c.GetLatestMagicBlock())
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

		if err = c.UpdateLatestMagicBlockFromSharders(ctx); err != nil {
			Logger.Error("update_mb_worker", zap.Error(err))
		}
	}

}

// ComputeBlockStateWithLock compute block state one by one
func (c *Chain) ComputeBlockStateWithLock(ctx context.Context, f func() error) (err error) {
	select {
	case c.computeBlockStateC <- struct{}{}:
		err = f()
		<-c.computeBlockStateC
	case <-ctx.Done():
		err = ctx.Err()
	}
	return
}
