package chain

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/httpclientutil"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/core/common"
	"0chain.net/core/config"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
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
	go c.SyncLFBTicketWorker(ctx)
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

			logging.Logger.Debug("[monitor] got new magic block, update nodes",
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

			logging.Logger.Info("[monitor] restart status monitor - new mb detected",
				zap.Int64("monitoring starting round", mb.StartingRound),
				zap.Int64("new mb starting round", cmb.StartingRound))
			cancel()
			mb = cmb
			cancel = startStatusMonitor(cmb, ctx)
		}
	}
}

func startStatusMonitor(mb *block.MagicBlock, ctx context.Context) func() {
	logging.Logger.Info("[monitor] start status monitor - update nodes",
		zap.Int64("mb starting round", mb.StartingRound))
	var smctx context.Context
	smctx, cancelCtx := context.WithCancel(ctx)
	waitMC := make(chan struct{})
	waitSC := make(chan struct{})
	go mb.Miners.StatusMonitor(smctx, mb.StartingRound, waitMC)
	go mb.Sharders.StatusMonitor(smctx, mb.StartingRound, waitSC)
	return func() {
		logging.Logger.Info("[monitor] cancel status monitor", zap.Int64("starting round", mb.StartingRound))
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
						logging.Logger.Warn("FinalizeRoundWorker finalize round timeout",
							zap.Int64("round", r.GetRoundNumber()))
						r.ResetFinalizingStateIfNotFinalized()
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

				logging.Logger.Debug("FinalizeRoundWorker - finalizing round slow, do fast moving",
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

// FinalizedBlockWorker - a worker that processes finalized blocks.
func (c *Chain) FinalizedBlockWorker(ctx context.Context, bsh BlockStateHandler) {
	for {
		select {
		case <-ctx.Done():
			return

		case fbr := <-c.finalizedBlocksChannel:
			func() {
				// TODO: make the timeout configurable
				timeout := c.ChainConfig.BlockFinalizationTimeout()
				cctx, cancel := context.WithTimeout(ctx, timeout)
				defer cancel()

				errC := make(chan error, 1)
				go func() {
					ts := time.Now()
					errC <- c.finalizeBlockProcess(cctx, fbr.block, bsh)
					logging.Logger.Debug("finalize block processed",
						zap.Int64("round", fbr.block.Round),
						zap.Duration("duration", time.Since(ts)))
				}()

				select {
				case err := <-errC:
					fbr.resultC <- err
				case <-cctx.Done():
					logging.Logger.Warn("finalize block process context done",
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
		logging.Logger.Warn("finalize block - slow finalized block processing",
			zap.Int64("lfb", lfb.Round), zap.Int64("fb", fb.Round))
	}

	if lfb.Round == fb.Round && lfb.Hash == fb.Hash {
		logging.Logger.Info("finalize block - already finalized",
			zap.Int64("round", fb.Round),
			zap.String("block", fb.Hash))
		return nil
	}

	logging.Logger.Debug("start to finalize block",
		zap.Int64("round", fb.Round),
		zap.String("block", fb.Hash),
		zap.String("prev block", fb.PrevHash))

	isSharder := node.Self.IsSharder()

	if !fb.IsStateComputed() {
		if fb.PrevBlock == nil {
			pb := c.GetLocalPreviousBlock(ctx, fb)
			if isSharder {
				if pb == nil || !pb.IsStateComputed() {
					logging.Logger.Error("finalize block - no previous block ready",
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
				logging.Logger.Error("finalize block - compute state failed",
					zap.Int64("round", fb.Round),
					zap.Error(err))
				return fmt.Errorf("compute state failed: %v", err)
			}
		} else {
			logging.Logger.Debug("finalize block - state not computed, try to fetch state changes",
				zap.Int64("round", fb.Round),
				zap.String("block", fb.Hash),
				zap.String("prev block", fb.PrevHash))

			if err := c.GetBlockStateChange(fb); err != nil {
				logging.Logger.Warn("finalize block failed to sync state from remote, try to compute state",
					zap.Int64("round", fb.Round),
					zap.Error(err))

				if err := c.ComputeState(ctx, fb); err != nil {
					logging.Logger.Error("finalize block - compute state failed",
						zap.Int64("round", fb.Round),
						zap.Error(err))
					return err
				}
				logging.Logger.Debug("finalize block - compute state success",
					zap.Int64("round", fb.Round),
					zap.String("block", fb.Hash))
			} else {
				logging.Logger.Debug("finalize block - sync state success",
					zap.Int64("round", fb.Round),
					zap.String("block", fb.Hash))
			}
		}
	}

	// TODO/TOTHINK: move the repair chain outside the finalized worker?
	// make sure we have valid verified MB chain if the block contains
	// a magic block; we already have verified and valid MB chain at this
	// moment, let's keep it updated and verified too

	// if isSharder {
	// get previous finalized block
	pr := c.GetRound(fb.Round - 1)
	if pr == nil {
		logging.Logger.Error("finalize block - previous round not found",
			zap.Int64("round", fb.Round))
		return errors.New("previous round is missing")
	}

	prevBlockHash := pr.GetBlockHash()
	if prevBlockHash == "" || !pr.IsFinalized() {
		logging.Logger.Error("finalize block - previous round not finalized",
			zap.Int64("round", fb.Round),
			zap.String("prev block", prevBlockHash),
			zap.Any("prev stat", pr.FinalizeState()))
		return errors.New("previous round not finalized")
	}

	if fb.PrevHash != prevBlockHash {
		logging.Logger.Error("finalize block - could not connect to lfb",
			zap.Int64("round", fb.Round),
			zap.String("block", fb.Hash),
			zap.String("prev block", fb.PrevHash),
			zap.String("finalized previous block", prevBlockHash))
		return errors.New("could not connect to lfb")
	}

	if err := c.finalizeBlock(ctx, fb, bsh); err != nil {
		return err
	}

	return c.postFinalize(ctx, fb)
}

/*PruneClientStateWorker - a worker that prunes the client state */
func (c *Chain) PruneClientStateWorker(ctx context.Context) {
	tick := 7 * time.Second
	timer := time.NewTimer(time.Second)
	logging.Logger.Debug("PruneClientStateWorker start")
	defer func() {
		logging.Logger.Debug("PruneClientStateWorker stopped, we should not see this...")
	}()

	for {
		select {
		case <-timer.C:
			logging.Logger.Debug("Do prune client state worker")
			c.pruneClientState(ctx)
			if c.pruneStats == nil {
				timer = time.NewTimer(time.Second)
			} else {
				timer = time.NewTimer(tick)
			}
		case <-ctx.Done():
			return
		}
	}
}

// SyncMissingNodes notify the nodes sync process to sync missing nodes
func (c *Chain) SyncMissingNodes(round int64, keys []util.Key, wc ...chan struct{}) {
	if len(keys) == 0 {
		return
	}
	go func() {
		for {
			select {
			case c.syncMissingNodesC <- syncPathNodes{
				round:  round,
				keys:   keys,
				replyC: wc,
			}:
				return
			case <-time.After(time.Second):
				logging.Logger.Debug("push to sync missing nodes channel timeout, retry...")
			}
		}
	}()
}

// SyncLFBStateWorker is a worker for syncing state of latest finalized round block.
// The worker would not sync state for every LFB as it will cause performance issue,
// only when it detects BC stuck will the synch process start.
func (c *Chain) SyncLFBStateWorker(ctx context.Context) {
	logging.Logger.Debug("SyncLFBStateWorker start")
	defer func() {
		logging.Logger.Debug("SyncLFBStateWorker stopped")
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

	// ticker to check if the BC is stuck
	tk := time.NewTicker(c.bcStuckCheckInterval)

	for {
		select {
		case bs := <-c.syncLFBStateC:
			// got a new finalized block summary
			if bs.Round > lastRound.round && lastRound.round > 0 {
				logging.Logger.Debug("BC is moving",
					zap.Int64("current_lfb_round", bs.Round),
					zap.Int64("last_round", lastRound.round))
				// update to latest finalized round
				lastRound.round = bs.Round
				lastRound.stateHash = bs.ClientStateHash
				lastRound.tm = time.Now()
				continue
			} else {
				logging.Logger.Debug("BC is not moving perhaps...")
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
				logging.Logger.Debug("last round tm < threashold...")
				continue
			}

			logging.Logger.Debug("BC may get stuck",
				zap.Int64("lastRound", lastRound.round),
				zap.String("state_hash", util.ToHex(lastRound.stateHash)),
				zap.Duration("stuck time", ts))

			// Try to recover by fetching latest magic block from miners
			if err := c.tryRecoverMagicBlock(ctx); err != nil {
				logging.Logger.Error("magic block recovery failed", zap.Error(err))
			}
		case mns := <-c.syncMissingNodesC:
			func() {
				var synced bool
				defer func() {
					for _, ch := range mns.replyC {
						if synced {
							select {
							case ch <- struct{}{}:
							default:
							}
						} else {
							close(ch)
						}
					}
				}()

				keysStr := make([]string, len(mns.keys))
				for i := range mns.keys {
					keysStr[i] = util.ToHex(mns.keys[i])
				}

				logging.Logger.Debug("sync missing nodes",
					zap.Int64("round", mns.round),
					zap.Strings("keys", keysStr))

				if err := c.GetStateNodes(ctx, mns.keys); err != nil {
					logging.Logger.Debug("sync missing nodes failed",
						zap.Int64("round", mns.round),
						zap.Strings("keys", keysStr),
						zap.Error(err))
					return
				}
				synced = true
			}()
		case <-ctx.Done():
			logging.Logger.Info("Context done, stop SyncLFBStateWorker")
			return
		}
	}
}

// tryRecoverMagicBlock attempts to fetch the latest magic block from miners
// and verify it by walking back to our current magic block.
func (c *Chain) tryRecoverMagicBlock(ctx context.Context) error {
	currentMB := c.GetCurrentMagicBlock()
	if currentMB == nil {
		return errors.New("no current magic block")
	}

	logging.Logger.Info("attempting magic block recovery",
		zap.Int64("current_mb_number", currentMB.MagicBlockNumber),
		zap.Int64("current_mb_starting_round", currentMB.StartingRound))

	// Get miners from current magic block to query
	miners := currentMB.Miners
	if miners == nil || miners.Size() == 0 {
		return errors.New("no miners in current magic block")
	}

	// Fetch latest finalized magic block from miners (returns the full block)
	latestBlock, err := c.fetchLatestMagicBlockFromMiners(ctx, miners)
	if err != nil {
		return fmt.Errorf("failed to fetch latest magic block: %v", err)
	}

	if latestBlock.MagicBlockNumber <= currentMB.MagicBlockNumber {
		logging.Logger.Debug("no newer magic block available",
			zap.Int64("current", currentMB.MagicBlockNumber),
			zap.Int64("fetched", latestBlock.MagicBlockNumber))
		return nil
	}

	logging.Logger.Info("found newer magic block",
		zap.Int64("current_mb", currentMB.MagicBlockNumber),
		zap.Int64("new_mb", latestBlock.MagicBlockNumber))

	// Verify by walking back from new MB to our current MB
	// Pass the full block so SetLatestFinalizedMagicBlock can be called
	if err := c.verifyMagicBlockChain(ctx, latestBlock.MagicBlock, currentMB, latestBlock, miners); err != nil {
		return fmt.Errorf("magic block chain verification failed: %v", err)
	}

	logging.Logger.Info("magic block recovery successful",
		zap.Int64("new_mb_number", latestBlock.MagicBlockNumber))

	return nil
}

// fetchLatestMagicBlockFromMiners fetches the latest finalized magic block from miners
// Returns the full block containing the magic block
func (c *Chain) fetchLatestMagicBlockFromMiners(ctx context.Context, miners *node.Pool) (*block.Block, error) {
	var (
		latestBlock *block.Block
		maxMBNumber int64
		mu          sync.Mutex
	)

	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		b, ok := entity.(*block.Block)
		if !ok || b.MagicBlock == nil {
			return nil, errors.New("invalid block entity")
		}

		mu.Lock()
		defer mu.Unlock()
		if b.MagicBlockNumber > maxMBNumber {
			maxMBNumber = b.MagicBlockNumber
			latestBlock = b
		}
		return b, nil
	}

	miners.RequestEntityFromAll(ctx, LatestFinalizedMagicBlockRequestor, nil, handler)

	if latestBlock == nil {
		return nil, errors.New("could not fetch magic block from any miner")
	}

	return latestBlock, nil
}

// verifyMagicBlockChain verifies the chain of magic blocks from newMB back to currentMB
// and also accepts the latest block that contains the newMB for proper LFMB update
func (c *Chain) verifyMagicBlockChain(ctx context.Context, newMB, currentMB *block.MagicBlock, latestBlock *block.Block, miners *node.Pool) error {
	// Collect all magic block-containing blocks from new to current by walking back
	// We need the full blocks to call SetLatestFinalizedMagicBlock later
	blockChain := []*block.Block{latestBlock}

	// Use sharders from the latest MB to fetch intermediate magic blocks
	// (sharders from old MB may not have newer MBs, but sharders from latest MB have all MBs)
	sharderURLs := newMB.Sharders.N2NURLs()
	logging.Logger.Info("verifying magic block chain",
		zap.Int64("from_mb", currentMB.MagicBlockNumber),
		zap.Int64("to_mb", newMB.MagicBlockNumber),
		zap.Int("sharder_count", len(sharderURLs)))

	// Walk back from newMB to currentMB (or further if needed due to fork)
	for blockChain[len(blockChain)-1].MagicBlock.MagicBlockNumber > currentMB.MagicBlockNumber+1 {
		prevMBNum := blockChain[len(blockChain)-1].MagicBlock.MagicBlockNumber - 1

		mb, err := httpclientutil.FetchMagicBlockFromSharders(ctx, sharderURLs, prevMBNum,
			func(b *block.Block) bool { return true }) // We'll verify the chain ourselves
		if err != nil {
			return fmt.Errorf("failed to fetch magic block %d: %v", prevMBNum, err)
		}

		if mb.MagicBlock == nil {
			return fmt.Errorf("magic block %d has no embedded magic block", prevMBNum)
		}

		// Verify hash chain
		expectedPrevHash := blockChain[len(blockChain)-1].MagicBlock.PreviousMagicBlockHash
		if mb.MagicBlock.Hash != expectedPrevHash {
			return fmt.Errorf("magic block %d hash mismatch: expected %s, got %s",
				prevMBNum, expectedPrevHash, mb.MagicBlock.Hash)
		}

		blockChain = append(blockChain, mb)
	}

	// Check if the chain connects to our current MB
	lastInChain := blockChain[len(blockChain)-1].MagicBlock
	if lastInChain.PreviousMagicBlockHash != currentMB.Hash {
		// The chain doesn't connect - our current MB might be from a fork
		// Fetch the correct MB from the network for the same number
		logging.Logger.Warn("local MB does not match network chain, fetching correct MB from network",
			zap.Int64("local_mb_number", currentMB.MagicBlockNumber),
			zap.String("local_mb_hash", currentMB.Hash),
			zap.String("expected_prev_hash", lastInChain.PreviousMagicBlockHash))

		// Continue walking back to include the correct version of current MB
		for {
			prevMBNum := blockChain[len(blockChain)-1].MagicBlock.MagicBlockNumber - 1
			if prevMBNum < 1 {
				// Reached genesis, apply all blocks
				logging.Logger.Info("walked back to genesis during fork recovery")
				break
			}

			mb, err := httpclientutil.FetchMagicBlockFromSharders(ctx, sharderURLs, prevMBNum,
				func(b *block.Block) bool { return true })
			if err != nil {
				return fmt.Errorf("failed to fetch magic block %d during fork recovery: %v", prevMBNum, err)
			}

			if mb.MagicBlock == nil {
				return fmt.Errorf("magic block %d has no embedded magic block during fork recovery", prevMBNum)
			}

			// Verify hash chain
			expectedPrevHash := blockChain[len(blockChain)-1].MagicBlock.PreviousMagicBlockHash
			if mb.MagicBlock.Hash != expectedPrevHash {
				return fmt.Errorf("magic block %d hash mismatch during fork recovery: expected %s, got %s",
					prevMBNum, expectedPrevHash, mb.MagicBlock.Hash)
			}

			blockChain = append(blockChain, mb)

			// Limit how far back we go (100 MBs should be enough to find common ancestor)
			if len(blockChain) > 100 {
				return fmt.Errorf("could not find common ancestor within 100 magic blocks")
			}
		}
	}

	// Apply magic blocks in order (from oldest to newest)
	// Call both UpdateMagicBlock and SetLatestFinalizedMagicBlock to update all pools
	for i := len(blockChain) - 1; i >= 0; i-- {
		b := blockChain[i]
		if err := c.UpdateMagicBlock(b.MagicBlock); err != nil {
			// Log but continue - UpdateMagicBlock may fail if MB is older than current
			logging.Logger.Debug("UpdateMagicBlock returned error (may be expected during fork recovery)",
				zap.Int64("mb_number", b.MagicBlockNumber),
				zap.Error(err))
		}
		// SetLatestFinalizedMagicBlock updates the LFMB channel that block fetcher uses
		c.SetLatestFinalizedMagicBlock(b)
		logging.Logger.Info("updated magic block",
			zap.Int64("mb_number", b.MagicBlockNumber),
			zap.Int64("starting_round", b.MagicBlock.StartingRound))
	}

	return nil
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
			logging.Logger.Debug("current lfmb number is greater than new lfmb number",
				zap.Int64("current_lfmb_number", currentLFMB.MagicBlockNumber),
				zap.Int64("new lfmb_number", latestMagicBlock.MagicBlockNumber),
				zap.Int64("current_lfmb_round", currentLFMB.Round),
				zap.Int64("new lfmb_round", latestMagicBlock.Round))
			return
		}

		if currentLFMB.MagicBlockNumber == latestMagicBlock.MagicBlockNumber {
			err = errors.New("verify chain history failed, latest magic block does not match")
			logging.Logger.Error("verify_chain_history failed",
				zap.Error(err),
				zap.String("current_lfmb_hash", currentLFMB.Hash),
				zap.String("latest_mb_hash", latestMagicBlock.Hash),
				zap.Int64("magic block number", currentLFMB.MagicBlockNumber))
			return
		}

		requestMBNum := currentLFMB.MagicBlockNumber + 1
		logging.Logger.Debug("verify_chain_history", zap.Int64("get_mb_number", requestMBNum))

		magicBlock, err = httpclientutil.FetchMagicBlockFromSharders(ctx, sharders, requestMBNum,
			func(b *block.Block) bool {
				return currentLFMB.VerifyMinersSignatures(b)
			})
		if err != nil {
			return common.NewError("get_lfmb_from_sharders",
				fmt.Sprintf("failed to get %d: %v", requestMBNum, err))
		}

		logging.Logger.Info("verify chain history",
			zap.Int64("mb_sr", magicBlock.StartingRound),
			zap.String("mb_hash", magicBlock.Hash),
			zap.Int64("mb_num", magicBlock.MagicBlockNumber))

		if err = c.UpdateMagicBlock(magicBlock.MagicBlock); err != nil {
			logging.Logger.Error("verify chain history - update magic block failed", zap.Error(err))
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

// SyncLFBTicketWorker - a worker that gets the latest finalized block from other sharders
// and bump the LFB ticket.
func (c *Chain) SyncLFBTicketWorker(ctx context.Context) {
	logging.Logger.Info("SyncLFBTicketWorker started")
	defer logging.Logger.Info("SyncLFBTicketWorker stopped")
	tk := time.NewTicker(time.Second)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tk.C:
			c.BumpLFBTicket(ctx)
		}
	}
}
