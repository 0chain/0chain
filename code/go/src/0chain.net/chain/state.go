package chain

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"

	"0chain.net/block"
	bcstate "0chain.net/chain/state"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/state"
	"0chain.net/transaction"
	"0chain.net/util"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//StateSaveTimer - a metric that tracks the time it takes to save the state
var StateSaveTimer metrics.Timer

//StateChangeSizeMetric - a metri  that tracks how many state nodes are changing with each block
var StateChangeSizeMetric metrics.Histogram

func init() {
	StateSaveTimer = metrics.GetOrRegisterTimer("state_save_timer", nil)
	StateChangeSizeMetric = metrics.NewHistogram(metrics.NewUniformSample(1024))
}

var ErrPreviousStateUnavailable = common.NewError("prev_state_unavailable", "Previous state not available")

//StateMismatch - indicate if there is a mismatch between computed state and received state of a block
const StateMismatch = "state_mismatch"

var ErrStateMismatch = common.NewError(StateMismatch, "Computed state hash doesn't match with the state hash of the block")

var ErrInsufficientBalance = common.NewError("insufficient_balance", "Balance not sufficient for transfer")

/*ComputeState - compute the state for the block */
func (c *Chain) ComputeState(ctx context.Context, b *block.Block) error {
	lock := b.StateMutex
	lock.Lock()
	defer lock.Unlock()
	return c.computeState(ctx, b)
}

//ComputeOrSyncState - try to compute state and if there is an error, just sync it
func (c *Chain) ComputeOrSyncState(ctx context.Context, b *block.Block) error {
	lock := b.StateMutex
	lock.Lock()
	defer lock.Unlock()
	err := c.computeState(ctx, b)
	if err != nil {
		bsc, err := c.getBlockStateChange(b)
		if err != nil {
			return err
		}
		if bsc != nil {
			c.applyBlockStateChange(b, bsc)
		}
		if !b.IsStateComputed() {
			Logger.Error("compute state - state change error", zap.Any("round", b.Round), zap.Any("block", b.Hash), zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *Chain) computeState(ctx context.Context, b *block.Block) error {
	if b.IsStateComputed() {
		return nil
	}
	pb := b.PrevBlock
	if pb == nil {
		c.GetPreviousBlock(ctx, b)
		pb = b.PrevBlock
		if pb == nil {
			b.SetStateStatus(block.StateFailed)
			if state.DebugBlock() {
				Logger.Error("compute state - previous block not available", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
			} else {
				if config.DevConfiguration.State {
					Logger.Error("compute state - previous block not available", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
				}
			}
			return ErrPreviousBlockUnavailable
		}
	}
	if !pb.IsStateComputed() {
		if pb.GetStateStatus() == block.StateFailed {
			c.GetBlockStateChange(pb)
			if !pb.IsStateComputed() {
				return ErrPreviousStateUnavailable
			}
		} else {
			Logger.Info("compute state - previous block state not ready", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Int8("prev_block_state", pb.GetBlockState()), zap.Int8("prev_block_state_status", pb.GetStateStatus()))
			err := c.ComputeState(ctx, pb)
			if err != nil {
				pb.SetStateStatus(block.StateFailed)
				if state.DebugBlock() {
					Logger.Error("compute state - error computing previous state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
				} else {
					if config.DevConfiguration.State {
						Logger.Error("compute state - error computing previous state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
					}
				}
				return err
			}
		}
	}
	if pb.ClientState == nil {
		if config.DevConfiguration.State {
			Logger.Error("compute state - previous state nil", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
		}
		return ErrPreviousStateUnavailable
	}
	b.SetStateDB(pb)
	Logger.Info("compute state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.String("begin_client_state", util.ToHex(b.ClientState.GetRoot())), zap.String("prev_block", b.PrevHash), zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)))
	for _, txn := range b.Txns {
		if datastore.IsEmpty(txn.ClientID) {
			txn.ComputeClientID()
		}
		if !c.UpdateState(b, txn) {
			if config.DevConfiguration.State {
				b.SetStateStatus(block.StateFailed)
				Logger.Error("compute state - update state failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.String("prev_block", b.PrevHash), zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)))
				return common.NewError("state_update_error", "error updating state")
			}
		}
	}
	if bytes.Compare(b.ClientStateHash, b.ClientState.GetRoot()) != 0 {
		b.SetStateStatus(block.StateFailed)
		if config.DevConfiguration.State {
			Logger.Error("compute state - state hash mismatch", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(b.ClientState.GetChangeCollector().GetChanges())), zap.String("block_state_hash", util.ToHex(b.ClientStateHash)), zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
		}
		return ErrStateMismatch
	}
	c.StateSanityCheck(ctx, b)
	Logger.Info("compute state successful", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(b.ClientState.GetChangeCollector().GetChanges())), zap.String("block_state_hash", util.ToHex(b.ClientStateHash)), zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
	b.SetStateStatus(block.StateSuccessful)
	return nil
}

//SaveChanges - persist the state changes
func (c *Chain) SaveChanges(ctx context.Context, b *block.Block) error {
	if !b.IsStateComputed() {
		err := c.ComputeOrSyncState(ctx, b)
		if err != nil {
			if config.DevConfiguration.State {
				Logger.Error("save changes - save state not successful", zap.Int64("round", b.Round), zap.String("hash", b.Hash), zap.Int8("state", b.GetBlockState()), zap.Error(err))
				if state.Debug() {
					Logger.DPanic("save changes - state not successful")
				}
			}
		}
	}
	if b.ClientState == nil {
		Logger.Error("save changes - client state is null", zap.Int64("round", b.Round), zap.String("hash", b.Hash))
		return nil
	}

	lock := b.StateMutex
	lock.Lock()
	defer lock.Unlock()
	var err error
	ts := time.Now()
	switch b.GetStateStatus() {
	case block.StateSynched:
		err = b.ClientState.SaveChanges(c.stateDB, false)
	case block.StateSuccessful:
		err = b.ClientState.SaveChanges(c.stateDB, false)
	default:
		return common.NewError("state_save_without_success", "State can't be saved without successful computation")
	}
	duration := time.Since(ts)
	StateSaveTimer.UpdateSince(ts)
	p95 := StateSaveTimer.Percentile(.95)
	changes := b.ClientState.GetChangeCollector().GetChanges()
	if len(changes) > 0 {
		StateChangeSizeMetric.Update(int64(len(changes)))
	}
	if StateSaveTimer.Count() > 100 && 2*p95 < float64(duration) {
		Logger.Error("save state - slow", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(changes)), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	} else {
		Logger.Debug("save state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(changes)), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration))
	}
	if err != nil {
		Logger.Error("save state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(changes)), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration), zap.Error(err))
	}
	return err
}

//GetBlockStateChange - get the state change of the block
func (c *Chain) GetBlockStateChange(b *block.Block) {
	bsc, err := c.getBlockStateChange(b)
	if err != nil {
		Logger.Error("get block state change - no bsc", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Error(err))
		return
	}
	if bsc == nil {
		return
	}
	Logger.Error("get block state change", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Int8("state_status", b.GetStateStatus()))
	err = c.ApplyBlockStateChange(b, bsc)
	if err != nil {
		Logger.Error("get block state change", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_hash", util.ToHex(b.ClientStateHash)), zap.Error(err))
	}
}

func (c *Chain) getBlockStateChange(b *block.Block) (*block.StateChange, error) {
	if b.PrevBlock == nil {
		return nil, ErrPreviousBlockUnavailable
	}
	if bytes.Compare(b.ClientStateHash, b.PrevBlock.ClientStateHash) == 0 {
		b.SetStateStatus(block.StateSynched)
		return nil, nil
	}
	bscRequestor := BlockStateChangeRequestor
	params := map[string]string{"block": b.Hash}
	ctx, cancelf := context.WithCancel(common.GetRootContext())
	var bsc *block.StateChange
	handler := func(ctx context.Context, entity datastore.Entity) (interface{}, error) {
		Logger.Debug("get block state change", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("bsc_id", entity.GetKey()))
		rsc, ok := entity.(*block.StateChange)
		if !ok {
			return nil, datastore.ErrInvalidEntity
		}
		if rsc.Hash != b.Hash {
			Logger.Error("get block state change - hash mismatch error", zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return nil, block.ErrBlockHashMismatch
		}
		root := rsc.GetRoot()
		if root == nil {
			Logger.Error("get block state change - state root error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("state_nodes", len(rsc.Nodes)))
			return nil, common.NewError("state_root_error", "Block state root calculcation error")
		}
		if bytes.Compare(b.ClientStateHash, root.GetHashBytes()) != 0 {
			Logger.Error("get block state change - state hash mismatch error", zap.Int64("round", b.Round), zap.String("block", b.Hash))
			return nil, block.ErrBlockStateHashMismatch
		}
		cancelf()
		bsc = rsc
		return rsc, nil
	}
	c.Miners.RequestEntity(ctx, bscRequestor, params, handler)
	if bsc == nil {
		return nil, common.NewError("block_state_change_error", "Error getting the block state change")
	}
	return bsc, nil
}

//ApplyBlockStateChange - apply the state chagnes to the block state
func (c *Chain) ApplyBlockStateChange(b *block.Block, bsc *block.StateChange) error {
	lock := b.StateMutex
	lock.Lock()
	defer lock.Unlock()
	return c.applyBlockStateChange(b, bsc)
}

func (c *Chain) applyBlockStateChange(b *block.Block, bsc *block.StateChange) error {
	if b.Hash != bsc.Hash {
		return block.ErrBlockHashMismatch
	}
	root := bsc.GetRoot()
	if root == nil {
		if b.PrevBlock != nil && bytes.Compare(b.PrevBlock.ClientStateHash, b.ClientStateHash) == 0 {
			return nil
		}
		return common.NewError("state_root_error", "state root not correct")
	}
	if bytes.Compare(b.ClientStateHash, root.GetHashBytes()) != 0 {
		return block.ErrBlockStateHashMismatch
	}
	if b.ClientState == nil {
		b.CreateState(bsc.GetNodeDB())
	}

	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()

	err := b.ClientState.MergeDB(bsc.GetNodeDB(), bsc.GetRoot().GetHashBytes())
	if err != nil {
		Logger.Error("apply block state change - error merging", zap.Int64("round", b.Round), zap.String("block", b.Hash))
	}
	b.SetStateStatus(block.StateSynched)
	return nil
}

func (c *Chain) rebaseState(lfb *block.Block) {
	if lfb.ClientState == nil {
		return
	}
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	ndb := lfb.ClientState.GetNodeDB()
	if ndb != c.stateDB {
		lfb.ClientState.SetNodeDB(c.stateDB)
		if lndb, ok := ndb.(*util.LevelNodeDB); ok {
			Logger.Debug("finalize round - rebasing current state db", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash), zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
			lndb.RebaseCurrentDB(c.stateDB)
			Logger.Debug("finalize round - rebased current state db", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash), zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
		}
	}
}

/*UpdateState - update the state of the transaction w.r.t the given block
* The block starts off with the state from the prior block and as transactions are processed into a block, the state gets updated
* If a state can't be updated (e.g low balance), then a false is returned so that the transaction will not make it into the block
 */
func (c *Chain) UpdateState(b *block.Block, txn *transaction.Transaction) bool {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	clientState := createTxnMPT(b.ClientState) // begin transaction
	startRoot := clientState.GetRoot()
	sctx := bcstate.NewStateContext(b, clientState, c.clientStateDeserializer, txn)

	switch txn.TransactionType {
	case transaction.TxnTypeData:
	case transaction.TxnTypeSend:
		sctx.AddTransfer(state.NewTransfer(txn.ClientID, txn.ToClientID, state.Balance(txn.Value)))
	}

	if err := sctx.Validate(); err != nil {
		return false
	}
	for _, transfer := range sctx.GetTransfers() {
		err := c.transferAmount(sctx, transfer.ClientID, transfer.ToClientID, state.Balance(transfer.Amount))
		if err != nil {
			return false
		}
	}

	err := b.ClientState.MergeMPTChanges(clientState) // commit transaction

	if err != nil {
		Logger.DPanic("update state - merge mpt error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
	}
	if state.DebugTxn() {
		if err := c.validateState(context.TODO(), b, startRoot); err != nil {
			Logger.DPanic("update state - state validation failure", zap.Any("txn", txn), zap.Error(err))
		}
		os, err := c.getState(b.ClientState, c.OwnerID)
		if err != nil || os == nil || os.Balance == 0 {
			Logger.DPanic("update state - owner account", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Any("os", os), zap.Error(err))
		}
	}
	return true
}

/*
* transferAmount - transfers balance from one account to another
*   when there is an error getting the state of the from or to account (other than no value), the error is simply returned back
*   when there is an error inserting/deleting the state of the from or to account, this results in fatal error when state is enabled
 */
func (c *Chain) transferAmount(sctx bcstate.StateContextI, fromClient, toClient datastore.Key, amount state.Balance) error {
	if amount == 0 {
		return nil
	}
	b := sctx.GetBlock()
	clientState := sctx.GetState()
	txn := sctx.GetTransaction()
	fs, err := c.getState(clientState, fromClient)
	if !isValid(err) {
		if config.DevConfiguration.State {
			Logger.Error("transfer amount - client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
		}
		if state.DebugTxn() {
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(stateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(stateOut, "transfer amount - error getting state value: %v %+v %v\n", fromClient, txn, err)
			printStates(clientState, b.ClientState)
			Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", fromClient, err))
		}
		return err
	}
	if fs.Balance < amount {
		return ErrInsufficientBalance
	}
	ts, err := c.getState(clientState, toClient)
	if !isValid(err) {
		if config.DevConfiguration.State {
			Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
		}
		if state.DebugTxn() {
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(stateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(stateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
			printStates(clientState, b.ClientState)
			Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", toClient, err))
		}
		return err
	}
	fs.SetRound(b.Round)
	fs.Balance -= amount
	if fs.Balance == 0 {
		Logger.Info("transfer amount - remove client", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client", fromClient), zap.Any("txn", txn))
		_, err = clientState.Delete(util.Path(fromClient))
	} else {
		_, err = clientState.Insert(util.Path(fromClient), fs)
	}
	if err != nil {
		if config.DevConfiguration.State {
			Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		if state.Debug() {
			Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
	}
	ts.SetRound(b.Round)
	ts.Balance += amount
	_, err = clientState.Insert(util.Path(toClient), ts)
	if err != nil {
		if config.DevConfiguration.State {
			Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		if state.Debug() {
			Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
	}
	return nil
}

func createTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion())
	tmpt.SetRoot(mpt.GetRoot())
	return tmpt
}

func (c *Chain) getState(clientState util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	if clientState == nil {
		return nil, common.NewError("get state", "client state does not exist")
	}
	s := &state.State{}
	s.Balance = state.Balance(0)
	ss, err := clientState.GetNodeValue(util.Path(clientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return s, err
	}
	s = c.clientStateDeserializer.Deserialize(ss).(*state.State)
	return s, nil
}

/*GetState - Get the state of a client w.r.t a block */
func (c *Chain) GetState(b *block.Block, clientID string) (*state.State, error) {
	return c.getState(b.ClientState, clientID)
}

func isValid(err error) bool {
	if err == nil {
		return true
	}
	if err == util.ErrValueNotPresent {
		return true
	}
	return false
}
