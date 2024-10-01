package sharder

import (
	"context"
	"errors"
	"fmt"
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
	defer ememorystore.CloseEntityConnection(rctx, roundEntityMetadata)
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
		logging.Logger.Error("load_lfb - verify notarization failed",
			zap.Error(err),
			zap.Int64("round", bl.lfb.Round),
			zap.String("block", bl.lfb.Hash))
		err = nil // not a real error
		return    // do nothing, if not notarized
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
		ememorystore.Close(rctx)
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
		if lfbr.Round <= lfbRound {
			// use LFB from state DB when:
			// LFB from state DB is more old than LFB from event DB or
			// They are in the same round
			lfbRound = lfbr.Round
			lfbHash = lfbr.Hash
		}
	}

	const maxRollbackRounds = 5
	var i int

loop:
	for {
		logging.Logger.Debug("load_lfb - load round and block",
			zap.Int64("round", lfbRound),
			zap.String("block", lfbHash))
		bl, err = sc.loadLFBRoundAndBlocks(ctx, lfbHash, lfbRound)
		if err != nil {
			return err
		}

		if bl.lfb.Round > sc.GetCurrentRound() {
			sc.SetCurrentRound(bl.lfb.Round)
		}

		logging.Logger.Debug("load_lfb, start to load latest finalized magic block from store")
		// and then, check out related LFMB can be missing
		bl.lfmb, err = sc.loadLatestFinalizedMagicBlockFromStore(ctx, bl.lfb)
		if err != nil {
			logging.Logger.Warn("load_lfb, missing corresponding lfmb",
				zap.Int64("round", bl.r.Number),
				zap.String("block_hash", bl.r.BlockHash),
				zap.String("lfmb_hash", bl.lfb.LatestFinalizedMagicBlockHash))
			// we can't skip to starting round, because we don't know it
			return err // the nil is 'use genesis'
		}

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
				logging.Logger.Error("load_lfb - check previous block",
					zap.Int64("round", lfbRound-1),
					zap.String("hash", bl.lfb.PrevHash))
				lfbRound = lfbRound - 1
				lfbHash = bl.lfb.PrevHash

				i++
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
	bl.nlfmb, err = sc.loadHighestMagicBlock(ctx, bl.lfb)
	if err != nil {
		logging.Logger.Warn("load_lfb, loading highest magic block", zap.Error(err))
	}

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
