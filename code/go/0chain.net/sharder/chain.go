package sharder

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"0chain.net/chaincore/transaction"
	"0chain.net/core/cache"
	"0chain.net/core/ememorystore"
	"github.com/0chain/common/core/logging"
	"github.com/0chain/common/core/util"
	"github.com/linxGnu/grocksdb"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/state"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/sharder/blockstore"

	"go.uber.org/zap"
)

const errInvalidStateCode = "invalid_state"

var errInvalidState = common.NewError(errInvalidStateCode, "")

var sharderChain = &Chain{}

/*SetupSharderChain - setup the sharder's chain */
func SetupSharderChain(c *chain.Chain) {
	sharderChain.Chain = c
	sharderChain.blockChannel = make(chan *block.Block, 1)
	sharderChain.RoundChannel = make(chan *round.Round, 1)
	blockCacheSize := 100
	sharderChain.BlockCache = cache.NewLRUCache[string, *block.Block](blockCacheSize)
	transactionCacheSize := 5 * blockCacheSize
	sharderChain.BlockTxnCache = cache.NewLRUCache[string, *transaction.TransactionSummary](transactionCacheSize)
	c.SetFetchedNotarizedBlockHandler(sharderChain)
	c.SetViewChanger(sharderChain)
	c.SetMagicBlockSaver(sharderChain)
	sharderChain.BlockSyncStats = &SyncStats{}
	c.RoundF = SharderRoundFactory{}
}

/*GetSharderChain - get the sharder's chain */
func GetSharderChain() *Chain {
	return sharderChain
}

/*Chain - A chain structure to manage the sharder activities */
type Chain struct {
	*chain.Chain
	blockChannel chan *block.Block
	// blockBuffer    *orderbuffer.OrderBuffer
	RoundChannel   chan *round.Round
	BlockCache     *cache.LRU[string, *block.Block]
	BlockTxnCache  *cache.LRU[string, *transaction.TransactionSummary]
	SharderStats   Stats
	BlockSyncStats *SyncStats
}

/*GetRoundChannel - get the round channel where the finalized rounds are put into for further processing */
func (sc *Chain) GetRoundChannel() chan *round.Round {
	return sc.RoundChannel
}

/*SetupGenesisBlock - setup the genesis block for this chain */
func (sc *Chain) SetupGenesisBlock(hash string, magicBlock *block.MagicBlock, initStates *state.InitStates) *block.Block {
	gr, gb := sc.GenerateGenesisBlock(hash, magicBlock, initStates)
	sc.AddRound(gr)
	sc.AddGenesisBlock(gb)

	// Save the round
	if err := sc.StoreRound(gr.(*round.Round)); err != nil {
		logging.Logger.Panic("setup genesis block, save genesis round failed", zap.Error(err))
	}

	// Save the block
	err := sc.storeBlock(gb)
	if err != nil {
		logging.Logger.Panic("setup genesis block, save genesis block failed", zap.Error(err))
	}

	if gb.MagicBlock != nil {
		var tries int64
		bs := gb.GetSummary()
		err = sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
		for err != nil {
			tries++
			logging.Logger.Error("setup genesis block -- failed to store magic block map", zap.Error(err), zap.Int64("tries", tries))
			time.Sleep(time.Millisecond * 100)
			err = sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
		}
	}
	return gb
}

/*GetBlockFromStore - get the block from the store */
func (sc *Chain) GetBlockFromStore(blockHash string, round int64) (*block.Block, error) {
	bs := block.BlockSummary{Hash: blockHash, Round: round}
	return sc.GetBlockFromStoreBySummary(&bs)
}

/*GetBlockFromStoreBySummary - get the block from the store */
func (sc *Chain) GetBlockFromStoreBySummary(bs *block.BlockSummary) (*block.Block, error) {
	b, err := blockstore.GetStore().ReadWithBlockSummary(bs)
	if err != nil {
		logging.Logger.Error("get block from store by summary failed", zap.Error(err))
		return nil, err
	}
	return b, nil
}

/*GetRoundFromStore - get the round from a store*/
func (sc *Chain) GetRoundFromStore(ctx context.Context, roundNum int64) (*round.Round, error) {
	r := datastore.GetEntity("round").(*round.Round)
	r.Number = roundNum
	roundEntityMetadata := r.GetEntityMetadata()
	rctx := ememorystore.WithEntityConnection(ctx, roundEntityMetadata)
	defer ememorystore.Close(rctx, roundEntityMetadata)
	err := r.Read(rctx, r.GetKey())
	return r, err
}

// GetBlockHash - get the block hash for a given round
func (sc *Chain) GetBlockHash(ctx context.Context, roundNumber int64) (string, error) {
	if roundNumber > sc.GetCurrentRound() {
		return "", fmt.Errorf("round %d does not exist", roundNumber)
	}

	var err error
	r := sc.GetSharderRound(roundNumber)
	if r == nil {
		r, err = sc.GetRoundFromStore(ctx, roundNumber)
		if err != nil {
			return "", err
		}
	}
	if r.BlockHash == "" {
		return "", fmt.Errorf("round %d has empty block hash", roundNumber)
	}
	return r.BlockHash, nil
}

// GetSharderRound - get the sharder's version of the round.
func (sc *Chain) GetSharderRound(roundNumber int64) *round.Round {
	r := sc.GetRound(roundNumber)
	if r == nil {
		return nil
	}
	sr, ok := r.(*round.Round)
	if !ok {
		return nil
	}
	return sr
}

type blocksLoaded struct {
	lfb   *block.Block // latest finalized block with stored client state
	lfmb  *block.Block // magic block related to the lfb
	r     *round.Round // round related to the lfb
	nlfmb *block.Block // magic block equal to the lfmb or newer
}

// verifyBlockContinuityFromStore walks blocks around candidateRound in local
// blockstore, verifies each has valid notarization, and returns the recommended
// LFB round (highest block with finalizationDepth+ notarized successors).
func (sc *Chain) verifyBlockContinuityFromStore(ctx context.Context, candidateRound int64, targetContinuity int, finalizationDepth int) (int64, error) {
	const minConsecutiveBlocks = 5 // need at least 5 consecutive valid blocks for LFB

	// Walk backward from candidateRound looking for a sequence of consecutive valid blocks.
	// When we hit an invalid block, reset the chain and keep looking further back.
	var chain []*block.Block

	for r := candidateRound; r > 0 && r > candidateRound-int64(targetContinuity); r-- {
		hash, err := sc.GetBlockHash(ctx, r)
		if err != nil {
			// No hash - reset chain and continue looking
			if len(chain) > 0 {
				logging.Logger.Debug("verify_continuity - gap in blocks, resetting chain",
					zap.Int64("round", r), zap.Int("chain_len", len(chain)))
			}
			chain = nil
			continue
		}
		b, err := sc.GetBlockFromStore(hash, r)
		if err != nil {
			// No block - reset chain and continue
			chain = nil
			continue
		}
		if err := sc.VerifyBlockNotarization(ctx, b); err != nil {
			logging.Logger.Debug("verify_continuity - block failed notarization, resetting chain",
				zap.Int64("round", r), zap.Error(err))
			// Invalid block - reset chain and continue looking
			chain = nil
			continue
		}

		// Valid block - prepend to chain
		chain = append([]*block.Block{b}, chain...)

		// Check if we have enough consecutive valid blocks
		if len(chain) >= minConsecutiveBlocks {
			logging.Logger.Debug("verify_continuity - found consecutive valid blocks",
				zap.Int("count", len(chain)),
				zap.Int64("from", chain[0].Round),
				zap.Int64("to", chain[len(chain)-1].Round))
			break
		}
	}

	if len(chain) < minConsecutiveBlocks {
		return 0, fmt.Errorf("could not find %d consecutive valid blocks within %d rounds back from %d (found %d)",
			minConsecutiveBlocks, targetContinuity, candidateRound, len(chain))
	}

	// Now walk forward from the end of our chain to extend it
	lastRound := chain[len(chain)-1].Round
	for r := lastRound + 1; len(chain) < targetContinuity; r++ {
		hash, err := sc.GetBlockHash(ctx, r)
		if err != nil {
			break // no more blocks
		}
		b, err := sc.GetBlockFromStore(hash, r)
		if err != nil {
			break
		}
		if err := sc.VerifyBlockNotarization(ctx, b); err != nil {
			break // chain ends at first invalid block going forward
		}
		chain = append(chain, b)
	}

	logging.Logger.Info("verify_continuity - continuous chain from local store",
		zap.Int("length", len(chain)),
		zap.Int64("from", chain[0].Round),
		zap.Int64("to", chain[len(chain)-1].Round))

	// Apply finalization depth: highest block with finalizationDepth successors
	if len(chain) > finalizationDepth {
		idx := len(chain) - 1 - finalizationDepth
		return chain[idx].Round, nil
	}

	// Not enough depth — use earliest block (safest)
	return chain[0].Round, nil
}

// fetchMissingBlocksFromSharders fetches blocks from peer sharders to fill gaps
// in local blockstore around the given round.
func (sc *Chain) fetchMissingBlocksFromSharders(ctx context.Context, aroundRound int64, count int) int {
	fetched := 0
	for r := aroundRound; r > aroundRound-int64(count) && r > 0; r-- {
		hash, err := sc.GetBlockHash(ctx, r)
		if err == nil {
			if _, berr := sc.GetBlockFromStore(hash, r); berr == nil {
				continue
			}
		}
		fetchCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		nb, nerr := sc.GetNotarizedBlockFromSharders(fetchCtx, "", r)
		cancel()
		if nerr != nil || nb == nil {
			continue
		}
		if serr := blockstore.GetStore().Write(nb); serr != nil {
			logging.Logger.Debug("fetch_missing - failed to store block",
				zap.Int64("round", r), zap.Error(serr))
			continue
		}
		fetched++
	}
	if fetched > 0 {
		logging.Logger.Info("fetch_missing - fetched blocks from sharders",
			zap.Int("fetched", fetched),
			zap.Int64("around_round", aroundRound))
	}
	return fetched
}

func (sc *Chain) setupLatestBlocks(ctx context.Context, bl *blocksLoaded) (
	err error) {

	// using ClientState of genesis block

	bl.lfb.SetStateStatus(block.StateSuccessful)
	if err = sc.InitBlockState(bl.lfb); err != nil {
		bl.lfb.SetStateStatus(0)
		logging.Logger.Error("load_lfb -- can't initialize stored block state",
			zap.Error(err))
		return common.NewErrorf(errInvalidStateCode, "can't init block state: %v", err) // fatal
	}

	// setup lfmb first
	if err = sc.UpdateMagicBlock(bl.lfmb.MagicBlock); err != nil {
		return common.NewErrorf("load_lfb",
			"can't update magic block: %v", err) // fatal
	}

	sc.SetRandomSeed(bl.r, bl.r.GetRandomSeed())
	bl.r.Finalize(bl.lfb)

	// set LFB and LFMB of the Chain, add the block to internal Chain's map
	sc.AddLoadedFinalizedBlocks(bl.lfb, bl.lfmb, bl.r)

	// check is it notarized
	err = sc.VerifyBlockNotarization(ctx, bl.lfb)
	if err != nil {
		logging.Logger.Error("load_lfb - verify notarization failed, triggering rollback",
			zap.Error(err),
			zap.Int64("round", bl.lfb.Round),
			zap.String("block", bl.lfb.Hash))
		// Return errInvalidState to trigger rollback to find a block with valid notarization
		return common.NewErrorf(errInvalidStateCode, "block notarization failed: %v", err)
	}
	// Verify block continuity and finalization depth
	recommendedLFB, cerr := sc.verifyBlockContinuityFromStore(ctx, bl.lfb.Round, 500, 3)
	if cerr != nil {
		logging.Logger.Warn("load_lfb - continuity check failed, proceeding with current LFB",
			zap.Int64("round", bl.lfb.Round), zap.Error(cerr))
		// Don't rollback — state is valid at current round, continuity check is advisory
	} else if recommendedLFB < bl.lfb.Round {
		logging.Logger.Info("load_lfb - finalization depth recommends earlier LFB, attempting switch",
			zap.Int64("current_lfb", bl.lfb.Round),
			zap.Int64("recommended_lfb", recommendedLFB))
		// Try to load the recommended block and its round from store
		recRound, roundErr := sc.GetRoundFromStore(ctx, recommendedLFB)
		if roundErr == nil {
			recBlock, blkErr := sc.GetBlockFromStore(recRound.BlockHash, recommendedLFB)
			if blkErr == nil {
				recBlock.SetStateStatus(block.StateSuccessful)
				if stErr := sc.InitBlockState(recBlock); stErr == nil {
					logging.Logger.Info("load_lfb - switched to recommended LFB",
						zap.Int64("round", recommendedLFB))
					bl.lfb = recBlock
					bl.r = recRound // IMPORTANT: also update the round!
					// Update chain's internal state to match the switched LFB
					sc.SetLatestFinalizedBlock(recBlock)
					sc.SetCurrentRound(recommendedLFB)
					// CRITICAL: Finalize the new round so subsequent rounds can be finalized
					sc.SetRandomSeed(recRound, recRound.GetRandomSeed())
					recRound.Finalize(recBlock)
					// CRITICAL: Update the round cache with the new LFB round
					sc.AddLoadedFinalizedBlocks(recBlock, bl.lfmb, recRound)
				} else {
					logging.Logger.Warn("load_lfb - recommended LFB has no state, keeping current",
						zap.Int64("recommended", recommendedLFB),
						zap.Int64("keeping", bl.lfb.Round),
						zap.Error(stErr))
				}
			}
		} else {
			logging.Logger.Warn("load_lfb - could not load recommended round from store, keeping current",
				zap.Int64("recommended", recommendedLFB),
				zap.Error(roundErr))
		}
	}

	bl.lfb.SetBlockNotarized()

	// add as notarized
	bl.lfb.SetBlockState(block.StateNotarized)
	bl.r.AddNotarizedBlock(bl.lfb)

	// setup nlfmb
	if bl.nlfmb != nil && bl.nlfmb.Round > bl.lfmb.Round {
		if err = sc.UpdateMagicBlock(bl.nlfmb.MagicBlock); err != nil {
			return common.NewErrorf("load_lfb",
				"can't update newer magic block: %v", err) // fatal
		}
		sc.SetLatestFinalizedMagicBlock(bl.nlfmb) // the real latest
	}

	return // everything is ok
}

func (sc *Chain) loadLatestFinalizedMagicBlockFromStore(ctx context.Context,
	lfb *block.Block) (lfmb *block.Block, err error) {

	// check out lfmb magic block hash

	if lfb.LatestFinalizedMagicBlockHash == "" {
		return nil, common.NewError("load_lfb",
			"empty LatestFinalizedMagicBlockHash field") // fatal or genesis
	}

	if lfb.LatestFinalizedMagicBlockHash == lfb.Hash {
		if lfb.MagicBlock == nil {
			// fatal
			return nil, common.NewError("load_lfb", "missing MagicBlock field")
		}
		return lfb, nil // the same
	}

	// load from store

	logging.Logger.Debug("load_lfb (lfmb) from store",
		zap.String("block_with_magic_block_hash",
			lfb.LatestFinalizedMagicBlockHash),
		zap.Int64("block_with_magic_block_round",
			lfb.LatestFinalizedMagicBlockRound))

	lfmb, err = blockstore.GetStore().Read(lfb.LatestFinalizedMagicBlockHash)
	if err != nil {
		// fatality, can't find related LFMB
		return nil, common.NewErrorf("load_lfb",
			"related magic block not found: hash: %v, err: %v", lfb.LatestFinalizedMagicBlockHash, err)
	}

	// with current implementation it's a case
	if lfmb == nil {
		// fatality, can't find related LFMB
		return nil, common.NewError("load_lfb",
			"related magic block not found (no error)")
	}

	logging.Logger.Debug("load_lfb (lfmb) from store", zap.Int64("round", lfmb.Round),
		zap.String("hash", lfmb.Hash))

	if lfmb.MagicBlock == nil {
		// fatal
		return nil, common.NewError("load_lfb", "missing MagicBlock field")
	}

	return
}

// just get highest known MB
func (sc *Chain) loadHighestMagicBlock(ctx context.Context,
	lfb *block.Block) (lfmb *block.Block, err error) {

	if lfb.MagicBlock != nil {
		return lfb, nil
	}

	var hmbm *block.MagicBlockMap
	if hmbm, err = sc.GetHighestMagicBlockMap(ctx); err != nil {
		return nil, common.NewErrorf("load_lfb",
			"getting highest MB map: %v", err) // critical
	}

	logging.Logger.Debug("load_lfb (lfmb), got round",
		zap.Int64("round", hmbm.BlockRound),
		zap.String("block_hash", hmbm.Hash))

	var bl *block.Block
	bl, err = sc.GetBlockFromStore(hmbm.Hash, hmbm.BlockRound)
	if err != nil {
		return nil, common.NewErrorf("load_lfb",
			"getting block with highest MB: %v", err) // critical
	}

	if bl.MagicBlock != nil {
		return bl, nil // got it
	}

	return // not found
}

func (sc *Chain) loadLatestMagicBlock(ctx context.Context) (lfmb *block.Block, err error) {
	var hmbm *block.MagicBlockMap
	if hmbm, err = sc.GetHighestMagicBlockMap(ctx); err != nil {
		return nil, common.NewErrorf("load_lfb",
			"getting highest MB map: %v", err) // critical
	}

	logging.Logger.Debug("load_lfb (lfmb), got round",
		zap.Int64("round", hmbm.BlockRound),
		zap.String("block_hash", hmbm.Hash))

	var bl *block.Block
	bl, err = sc.GetBlockFromStore(hmbm.Hash, hmbm.BlockRound)
	if err != nil {
		return nil, common.NewErrorf("load_lfb",
			"getting block with highest MB: %v", err) // critical
	}

	if bl.MagicBlock != nil {
		return bl, nil // got it
	}

	return // not found
}

func (sc *Chain) LoadLatestMBs(ctx context.Context, fromMBNumber int64) (mbs []*block.Block) {
	// iterate from fromMBNumber back 5 or till 1,
	var count = 5
	mbs = make([]*block.Block, 0, count)
	for i := fromMBNumber; i > 0 && count > 0; i-- {
		count--
		mbStr := strconv.FormatInt(i, 10)
		mb, err := sc.GetMagicBlockMap(ctx, mbStr)
		if err != nil {
			logging.Logger.Error("load_latest_mb", zap.Error(err), zap.Int64("mb number", i))
			continue
		}
		logging.Logger.Info("load_latest_mb load latest MB from store", zap.Int64("mb number", fromMBNumber))
		b, err := sc.GetBlockFromHash(ctx, mb.Hash, mb.BlockRound)
		if err != nil {
			logging.Logger.Error("load_latest_mb failed to load block from store",
				zap.Error(err),
				zap.Int64("mb number", i),
				zap.Int64("round", mb.BlockRound),
				zap.String("hash", mb.Hash))

			// try to fetch from remote
			b, err = sc.GetNotarizedBlockFromSharders(ctx, mb.Hash, mb.BlockRound)
			if err != nil {
				logging.Logger.Error("load_latest_mb failed to load block from remote",
					zap.Error(err),
					zap.Int64("mb number", i),
					zap.Int64("round", mb.BlockRound),
					zap.String("hash", mb.Hash))
				continue
			}

			logging.Logger.Info("load_latest_mb loaded block from remote",
				zap.Int64("mb number", i),
				zap.Int64("round", mb.BlockRound),
				zap.String("hash", mb.Hash))
		}
		mbs = append(mbs, b)
	}

	return mbs
}

func (sc *Chain) LoadLatestFinalizedMagicBlockFromStore(ctx context.Context) {
	lfmb := sc.GetLatestMagicBlock()
	// load the latest N magic blocks
	n := int64(5) // TODO: read from config
	retry := 3

	if lfmb.MagicBlockNumber <= 1 {
		return
	}

	// magic block number start from 1, the genesis block
	startNum := int64(2) // 1 is the genesis block, we have it locally, so don't need to fetch from remote
	if lfmb.MagicBlockNumber < startNum {
		// genesis block, return
		return
	}

	newStart := lfmb.MagicBlockNumber - n
	if newStart > startNum {
		startNum = newStart
	}

	for i := startNum; i <= lfmb.MagicBlockNumber; i++ {
		// load MB from local store
		mbStr := strconv.FormatInt(i, 10)
		prevMbStr := strconv.FormatInt(i-1, 10)
		mb, err := block.LoadMagicBlock(ctx, mbStr)
		if err != nil {
			logging.Logger.Panic("load_latest_mb", zap.Error(err), zap.Int64("mb number", i))
		}

		var prevMb *block.MagicBlock
		if i == 2 {
			// previous magic block is the genesis block
			prevMb = sc.GetMagicBlock(1)
		} else {
			prevMb, err = block.LoadMagicBlock(ctx, prevMbStr)
			if err != nil {
				logging.Logger.Panic("load_latest_mb", zap.Error(err), zap.Int64("mb number", i))
			}
		}

		sc.InitializeMinerPoolIfNotSet(prevMb)

		// load and set prev mb if not in chain.MagicBlockStorage so that
		// blocks fetch process can verify tickets
		// if mc.MagicBlockStorage.GetByStartingRound(prevMb.StartingRound) == nil {
		sc.MagicBlockStorage.Put(prevMb, prevMb.StartingRound)
		// } else {
		// 	logging.Logger.Error("[mvc] load prev MB by magic bock number",
		// 		zap.Int64("mb number", i),

		// }

		logging.Logger.Info("[mvc] load MB by magic bock number", zap.Int64("mb number", i))
		for j := 0; j < retry; j++ {
			bmb, err := sc.GetNotarizedBlockFromSharders(ctx, "", mb.StartingRound)
			if err != nil {
				logging.Logger.Error("load_lfb - could not fetch latest finalized magic block from sharders",
					zap.Int64("mb_starting_round", lfmb.StartingRound), zap.Error(err))
				time.Sleep(3 * time.Second)
				continue
			}
			sc.UpdateMagicBlocks(bmb)
			break
		}
	}
}

func (sc *Chain) walkDownLookingForLFB(iter *grocksdb.Iterator, r *round.Round) (lfb *block.Block, err error) {

	var rollBackCount int
	for ; iter.Valid(); iter.Prev() {
		if rollBackCount >= sc.PruneStateBelowCount() {
			// could not recovery as the state of round below prune count may have nodes missing, and
			// we can not sync from remote neither, so just panic.
			if lfb != nil {
				logging.Logger.Panic("load_lfb, could not rollback to LFB with full state, please clean DB and sync again",
					zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash), zap.String("round_block_hash", r.BlockHash),
					zap.Int64("round_number", r.GetRoundNumber()))
			} else {
				logging.Logger.Panic("load_lfb, could not rollback to LFB with full state, please clean DB and sync again",
					zap.String("round_block_hash", r.BlockHash), zap.Int64("round_number", r.GetRoundNumber()))
			}
		}

		if err = datastore.FromJSON(iter.Value().Data(), r); err != nil {
			return nil, common.NewErrorf("load_lfb",
				"decoding round info: %v", err) // critical
		}

		logging.Logger.Debug("load_lfb, got round", zap.Int64("round", r.Number),
			zap.String("block_hash", r.BlockHash))

		lfb, err = sc.GetBlockFromStore(r.BlockHash, r.Number)
		if err != nil {
			logging.Logger.Error("load_lfb, could not get block from store", zap.Error(err))
			rollBackCount++
			continue // TODO: can we use os.IsNotExist(err) or should not
		}

		logging.Logger.Debug("load_lfb, got block", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash))

		lfnb, er := func() (*block.Block, error) {
			logging.Logger.Debug("load_lfb - get notarized block from sharders")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return sc.GetNotarizedBlockFromSharders(ctx, "", lfb.Round)
		}()

		if er != nil {
			logging.Logger.Warn("load_lfb, could not sync LFB from remote",
				zap.Int64("round", lfb.Round),
				zap.String("lfb", lfb.Hash))

			rollBackCount++
			continue
			// return
		}

		logging.Logger.Debug("load_lfb, got notarized block from remote and compare with local")

		if lfnb.Hash != lfb.Hash {
			logging.Logger.Warn("load_lfb, see different lfb, roll back",
				zap.Int64("round", lfb.Round),
				zap.String("local lfb", lfb.Hash),
				zap.String("remote lfb", lfnb.Hash))
			rollBackCount++
			continue
		}

		// check out required corresponding state

		// Don't check the state. It can be missing if the state had synced.
		// But it works fine anyway.

		logging.Logger.Debug("load_lfb, check if LFB state is in store")
		if !sc.HasClientStateStored(lfb.ClientStateHash) {
			logging.Logger.Warn("load_lfb, missing corresponding state",
				zap.Int64("round", r.Number),
				zap.String("block_hash", r.BlockHash))
			// we can't use this block, because of missing or malformed state
			rollBackCount++
			continue
		}

		go func() {
			// check if lfb has full state and sync all missing nodes
			if !sc.ValidateState(lfb) {
				logging.Logger.Warn("load_lfb, lfb state missing nodes",
					zap.Int64("round", r.Number),
					zap.String("block_hash", r.BlockHash))
			}
		}()

		logging.Logger.Debug("load_lfb, find it", zap.Int64("round", lfb.Round))
		return // got it
	}

	return nil, common.NewError("load_lfb", "no valid lfb found")
}

func (sc *Chain) loadLFBRoundAndBlocks(ctx context.Context, hash string, round int64) (*blocksLoaded, error) {
	r, err := sc.GetRoundFromStore(ctx, round)
	if err != nil {
		return nil, fmt.Errorf("load_lfb - could not load round from store: %v", err)
	}
	if r.BlockHash != hash {
		return nil, errors.New("load_lfb - block hash does not match")
	}

	lfb, err := sc.GetBlockFromStore(r.BlockHash, r.Number)
	if err != nil {
		logging.Logger.Error("load_lfb, could not get block from store", zap.Error(err))

		// get from remote
		var err error
		lfb, err = func() (*block.Block, error) {
			logging.Logger.Debug("load_lfb - get notarized block from sharders")
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			return sc.GetNotarizedBlockFromSharders(ctx, r.BlockHash, r.Number)
		}()

		if err != nil {
			return nil, fmt.Errorf("load_lfb - could not load lfb block: %v", err)
		}
	}

	logging.Logger.Debug("load_lfb, got block", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash))

	lfnb, err := func() (*block.Block, error) {
		logging.Logger.Debug("load_lfb - get notarized block from sharders")
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		return sc.GetNotarizedBlockFromSharders(ctx, "", lfb.Round)
	}()

	bl := blocksLoaded{
		r:   r,
		lfb: lfb,
	}

	if err != nil {
		logging.Logger.Debug("load_lfb, could not sync LFB from remote, use local LFB",
			zap.Int64("round", lfb.Round),
			zap.String("lfb", lfb.Hash),
			zap.Error(err))

		return &bl, nil
	}

	logging.Logger.Debug("load_lfb, got notarized block from remote and compare with local")

	if lfnb.Hash != lfb.Hash {
		logging.Logger.Warn("load_lfb, see different lfb",
			zap.Int64("round", lfb.Round),
			zap.String("local lfb", lfb.Hash),
			zap.String("remote lfb", lfnb.Hash))
		return nil, errors.New("load_lfb - see different lfb and notarized lfb")
	}

	return &bl, nil
}

// iterate over rounds from latest to zero looking for LFB and ignoring
// missing blocks in blockstore
func (sc *Chain) iterateRoundsLookingForLFB(ctx context.Context) *blocksLoaded {
	var (
		bl   = new(blocksLoaded)
		remd = datastore.GetEntityMetadata("round")
		rctx = ememorystore.WithEntityConnection(ctx, remd)
		conn = ememorystore.GetEntityCon(rctx, remd)
		iter = conn.Conn.NewIterator(conn.ReadOptions)

		// the error is internal, we are using logs and rolling back to
		// genesis blocks on error
		err error
	)

	defer func() {
		ememorystore.Close(rctx, remd)
		iter.Close()
	}()

	bl.r = remd.Instance().(*round.Round) //

	iter.SeekToLast() // from last

	if !iter.Valid() {
		return nil // the nil is 'use genesis'
	}

	if bl.lfb, err = sc.walkDownLookingForLFB(iter, bl.r); err != nil {
		logging.Logger.Warn("load_lfb, can't load lfb",
			zap.Int64("round_stopped", bl.r.Number),
			zap.Error(err))
		return nil // the nil is 'use genesis'
	}

	logging.Logger.Debug("load_lfb, finish walk down looking")
	return bl
}

// LoadLatestBlocksFromStore loads LFB and LFMB from store and sets them
// to corresponding fields of the sharder's Chain.
func (sc *Chain) LoadLatestBlocksFromStore(ctx context.Context) (err error) {
	lfbHash, lfbRound, err := sc.GetLatestFinalizedBlockFromDB()
	if err != nil {
		logging.Logger.Panic("Error getting latest finalized block from db", zap.Error(err))
		return err
	}

	if lfbRound == 0 {
		// use genesis
		logging.Logger.Debug("load_lfb - load from event db, use genesis block")
		// Mark loading complete even for genesis fallback
		sc.Chain.SetLFBLoadingComplete()
		return nil
	}

	logging.Logger.Debug("load_lfb - load from event db",
		zap.Int64("round", lfbRound),
		zap.String("block", lfbHash))

	var bl *blocksLoaded
	lfbr, err := sc.LoadLFBRound()
	if err != nil {
		// use the LFB info from event db
		logging.Logger.Warn("load_lfb - could not load lfb from state DB, continue using the one from event db",
			zap.Int64("lfb_round", lfbRound),
			zap.String("lfb_hash", lfbHash),
			zap.Error(err))
	} else {
		logging.Logger.Debug("load_lfb - load from stateDB",
			zap.Int64("round", lfbr.Round),
			zap.String("block", lfbr.Hash))
		// load and set up latest magic block
		mbs := sc.LoadLatestMBs(ctx, lfbr.MagicBlockNumber)
		if len(mbs) != 0 {
			// Store all MBs in the MagicBlockStorage
			for i := len(mbs) - 1; i >= 0; i-- {
				sc.SetMagicBlock(mbs[i].MagicBlock)
			}

			// Determine which MB to use as LFMB based on LFB round from state DB.
			// The LFMB starting round should not exceed the LFB round, otherwise
			// block validation will fail for blocks between LFB and LFMB because
			// the miner pool won't include all miners that created those blocks.
			selectedMB := mbs[0]
			if mbs[0].MagicBlock != nil && mbs[0].MagicBlock.StartingRound > lfbr.Round {
				logging.Logger.Warn("load_lfb - LFMB starting round ahead of LFB, finding appropriate MB",
					zap.Int64("lfmb_sr", mbs[0].MagicBlock.StartingRound),
					zap.Int64("lfb_round", lfbr.Round),
					zap.Int64("lfmb_number", mbs[0].MagicBlock.MagicBlockNumber))
				// Find the MB whose starting round is <= lfbr.Round (state DB)
				for i := 1; i < len(mbs); i++ {
					if mbs[i].MagicBlock != nil && mbs[i].MagicBlock.StartingRound <= lfbr.Round {
						logging.Logger.Info("load_lfb - using appropriate MB for LFB round",
							zap.Int64("lfb_round", lfbr.Round),
							zap.Int64("mb_sr", mbs[i].MagicBlock.StartingRound),
							zap.Int64("mb_number", mbs[i].MagicBlock.MagicBlockNumber))
						selectedMB = mbs[i]
						break
					}
				}
			}

			// Set PreviousMagicBlock BEFORE calling UpdateMagicBlock so that SetupNodes
			// includes nodes from the previous MB. This is critical during MB transitions
			// where a miner may be removed from the new MB but their blocks still need
			// to be validated.
			if selectedMB.MagicBlock != nil && selectedMB.MagicBlock.MagicBlockNumber > 1 {
				// Look for the previous MB in the loaded MBs slice
				for i := 0; i < len(mbs); i++ {
					if mbs[i].MagicBlock != nil &&
						mbs[i].MagicBlock.MagicBlockNumber == selectedMB.MagicBlock.MagicBlockNumber-1 {
						sc.Chain.PreviousMagicBlock = mbs[i].MagicBlock
						logging.Logger.Debug("load_lfb - set previous MB for node registration",
							zap.Int64("prev_mb_number", mbs[i].MagicBlock.MagicBlockNumber),
							zap.Int64("prev_mb_sr", mbs[i].MagicBlock.StartingRound),
							zap.Int("prev_miners", mbs[i].MagicBlock.Miners.Size()))
						break
					}
				}
			}

			// Now set up nodes and LFMB with the selected MB
			sc.UpdateMagicBlock(selectedMB.MagicBlock)
			sc.SetLatestFinalizedMagicBlock(selectedMB)

			if lfbr.Round <= lfbRound {
				// use LFB from state DB when:
				// LFB from state DB is more old than LFB from event DB or
				// They are in the same round
				lfbRound = lfbr.Round
				lfbHash = lfbr.Hash
			}

		} else if sc.IsViewChangeEnabled() {
			logging.Logger.Error("load_lfb - could not load latest magic block")
			return common.NewError("load_lfb", "could not see any latest magic block in local store")
		} else {
			logging.Logger.Warn("load_lfb - could not load latest magic block")
		}

	}

	// lfmb, err := sc.loadLatestMagicBlock(ctx)
	// if err != nil {
	// 	logging.Logger.Error("load_lfb, could not loading highest magic block", zap.Error(err))
	// 	return
	// }

	// sc.UpdateMagicBlock(lfmb.MagicBlock)

	const maxRollbackRounds = 500
	const maxLocalFailsBeforeNetworkFetch = 10
	var i int

loop:
	for {
		logging.Logger.Debug("load_lfb, start to load latest finalized magic block from store")
		// and then, check out related LFMB can be missing
		// sc.LoadLatestFinalizedMagicBlockFromStore(ctx)

		logging.Logger.Debug("load_lfb - load round and block",
			zap.Int64("round", lfbRound),
			zap.String("block", lfbHash))
		bl, err = sc.loadLFBRoundAndBlocks(ctx, lfbHash, lfbRound)
		if err != nil {
			return err
		}

		// Set current round to LFB round (handles both forward sync and rollback)
		if bl.lfb.Round != sc.GetCurrentRound() {
			sc.SetCurrentRound(bl.lfb.Round)
		}

		bl.lfmb = sc.GetLatestFinalizedMagicBlock(ctx)

		// setup all related for a non-genesis case
		err := sc.setupLatestBlocks(ctx, bl)
		switch err {
		case nil:
			break loop
		default:
			logging.Logger.Error("load_lfb - setup latest blocks failed", zap.Error(err))
			if lfbRound == 0 {
				return err
			}

			cerr, ok := err.(*common.Error)
			if ok && cerr.Is(errInvalidState) {
				i++

				// After maxLocalFailsBeforeNetworkFetch consecutive local failures,
				// stop rolling back and try to fetch LFB from peer sharders
				if i == maxLocalFailsBeforeNetworkFetch {
					logging.Logger.Warn("load_lfb - too many local failures, fetching LFB from peer sharders",
						zap.Int("failures", i),
						zap.Int64("current_round", lfbRound))

					fetchCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					nb, nerr := sc.GetNotarizedBlockFromSharders(fetchCtx, "", lfbRound)
					cancel()
					if nerr == nil && nb != nil {
						logging.Logger.Info("load_lfb - fetched block from peer sharders, retrying",
							zap.Int64("round", nb.Round),
							zap.String("block", nb.Hash))
						// Store the fetched block locally
						if serr := blockstore.GetStore().Write(nb); serr != nil {
							logging.Logger.Warn("load_lfb - failed to store fetched block",
								zap.Int64("round", nb.Round), zap.Error(serr))
						}
						lfbRound = nb.Round
						lfbHash = nb.Hash
						continue
					}
					logging.Logger.Warn("load_lfb - failed to fetch from peer sharders",
						zap.Int64("round", lfbRound), zap.Error(nerr))
				}

				logging.Logger.Error("load_lfb - check previous block",
					zap.Int64("round", lfbRound-1),
					zap.String("hash", bl.lfb.PrevHash))
				lfbRound = lfbRound - 1
				lfbHash = bl.lfb.PrevHash

				if i >= maxRollbackRounds {
					logging.Logger.Error("load_lfb - rollback max count meet", zap.Int("max", maxRollbackRounds))

					bl = sc.iterateRoundsLookingForLFB(ctx)
					if bl != nil {
						logging.Logger.Debug("load_lfb - iterate rounds looking for lfb",
							zap.Int64("round", bl.lfb.Round),
							zap.String("block", bl.lfb.Hash))
						lfbRound = bl.lfb.Round
						lfbHash = bl.lfb.Hash
						continue
					}

					return err
				}

				continue
			}
			return err
		}
	}

	magicBlockMiners := sc.GetMiners(bl.r.GetRoundNumber())
	bl.r.SetRandomSeedForNotarizedBlock(bl.lfb.GetRoundRandomSeed(), magicBlockMiners.Size())

	// if err := sc.setupLatestBlocks(ctx, bl); err != nil {
	// 	logging.Logger.Error("load_lfb - setup latest blocks failed", zap.Error(err))
	// 	return err
	// }

	// but the lfmb can be less than real latest finalized magic block,
	// the lfmb is just magic block related to the lfb, for example for
	// 502 round lfmb is 251, but lfmb of 501 round we already have and
	// it is the latest magic block, we have to load it and setup

	// using another round instance
	// bl.nlfmb, err = sc.loadHighestMagicBlock(ctx, bl.lfb)
	// if err != nil {
	// 	logging.Logger.Warn("load_lfb, loading highest magic block", zap.Error(err))
	// }

	// return bl // got them all (or excluding the nlfmb)

	// if bl == nil || bl.r == nil || bl.r.Number == 0 || bl.r.Number == 1 {
	// 	logging.Logger.Debug("load_lfb, use genesis block")
	// 	return // use genesis blocks
	// }

	logging.Logger.Debug("load_lfb from store",
		zap.Int64("round", bl.lfb.Round),
		zap.String("hash", bl.lfb.Hash),
		zap.Int64("lfmb", bl.lfmb.Round))

	if bl.nlfmb != nil && bl.nlfmb.Round != bl.lfmb.Round {
		logging.Logger.Debug("load_lfb from store (nlfmb)",
			zap.Int64("round", bl.nlfmb.Round))
	}

	// Reset LFB ticket to match actual LFB after any rollback during startup
	sc.Chain.ResetLFBTicket(ctx, bl.lfb)

	// Mark LFB loading as complete - workers can now call BumpLFBTicket
	// and LFBTicketHandler can accept network tickets
	sc.Chain.SetLFBLoadingComplete()

	return nil
}

// SaveMagicBlockHandler used on sharder startup to save received
// magic blocks. It's required to be able to load previous state.
func (sc *Chain) SaveMagicBlockHandler(ctx context.Context,
	b *block.Block) (err error) {

	logging.Logger.Info("save received magic block verifying chain",
		zap.Int64("round", b.Round), zap.String("hash", b.Hash),
		zap.Int64("starting_round", b.MagicBlock.StartingRound),
		zap.String("mb_hash", b.MagicBlock.Hash))

	if err = sc.storeBlock(b); err != nil {
		return
	}
	var bs = b.GetSummary()
	return sc.StoreMagicBlockMapFromBlock(bs.GetMagicBlockMap())
}

// SaveMagicBlock function.
func (sc *Chain) SaveMagicBlock() chain.MagicBlockSaveFunc {
	return chain.MagicBlockSaveFunc(sc.SaveMagicBlockHandler)
}

func (sc *Chain) ValidateState(b *block.Block) bool {
	logging.Logger.Debug("load_lfb, validate state - init state DB")
	if err := b.InitStateDB(sc.GetStateDB()); err != nil {
		logging.Logger.Warn("load_lfb, init block state failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.String("state", util.ToHex(b.ClientStateHash)),
			zap.Error(err))

		return false
	}

	logging.Logger.Debug("load_lfb, sync missing nodes")
	if err := sc.syncLFBMissingNodes(b); err != nil {
		logging.Logger.Warn("load_lfb, sync missing nodes failed",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Error(err))
		return false
	}

	logging.Logger.Debug("load_lfb, alidate state - sync msissing nodes done")
	return true
}

func (sc *Chain) syncLFBMissingNodes(b *block.Block) error {
	for {
		missing, err := b.ClientState.HasMissingNodes(context.Background())
		if err != nil {
			logging.Logger.Warn("load_lfb, find missing nodes failed",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.Error(err))
			return err
		}

		if !missing {
			return nil
		}

		keys := b.ClientState.GetMissingNodeKeys()
		keysStr := make([]string, len(keys))
		for i := range keys {
			keysStr[i] = util.ToHex(keys[i])
		}
		logging.Logger.Warn("load_lfb, lfb sync missing nodes",
			zap.Int64("round", b.Round),
			zap.Any("missing nodes", keysStr),
			zap.String("block", b.Hash))

		if err := sc.GetStateNodes(context.Background(), keys); err != nil {
			logging.Logger.Warn("load_lfb, sync missing nodes failed", zap.Error(err))
			return err
		}
	}
}
