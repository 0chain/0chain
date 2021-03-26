package chain

import (
	"bytes"
	"context"
	"fmt"
	"math"
	"time"

	"0chain.net/chaincore/block"
	bcstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
	"errors"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//StateSaveTimer - a metric that tracks the time it takes to save the state
var StateSaveTimer metrics.Timer

//StateChangeSizeMetric - a metric that tracks how many state nodes are changing with each block
var StateChangeSizeMetric metrics.Histogram

//SmartContractExecutionTimer - a metric that tracks the time it takes to execute a smart contract txn
var SmartContractExecutionTimer metrics.Timer

func init() {
	StateSaveTimer = metrics.GetOrRegisterTimer("state_save_timer", nil)
	StateChangeSizeMetric = metrics.NewHistogram(metrics.NewUniformSample(1024))

	SmartContractExecutionTimer = metrics.GetOrRegisterTimer("sc_execute_timer", nil)
}

var (
	ErrPreviousStateUnavailable = common.NewError("prev_state_unavailable", "Previous state not available")
	ErrPreviousStateNotComputed = common.NewError("prev_state_not_computed", "Previous state not computed")
)

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

// ComputeOrSyncState - try to compute state and if there is an error, just sync it
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
			if err = c.applyBlockStateChange(b, bsc); err != nil {
				Logger.Error("compute state - applying state change",
					zap.Any("round", b.Round), zap.Any("block", b.Hash),
					zap.Error(err))
				return err
			}
		}
		if !b.IsStateComputed() {
			Logger.Error("compute state - state change error",
				zap.Any("round", b.Round), zap.Any("block", b.Hash),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *Chain) updateStateFromNetwork(ctx context.Context, b *block.Block) error {
	if b.IsStateComputed() {
		return nil
	}

	blocks := make([]*block.Block, 0, 50)
	blocks = append(blocks, b)
	lfr := c.GetLatestFinalizedBlock().Round

	round := b.Round
	if lfr == round {
		Logger.Error("Finalized block is not computed")
		return errors.New("finalized block is not computed")
	}

	for r := round - 1; r >= lfr; r-- {
		rd := c.GetRound(r)
		if rd == nil {
			Logger.Error("Round does not exist",
				zap.Int64("round", r),
				zap.Int64("current_round", round),
				zap.Int64("latest_finalized_round", lfr))
			return fmt.Errorf("round does not exist, round: %d, current_round: %d, latest_determinisitc_round: %d", r, round, lfr)
		}

		pb := rd.GetHeaviestNotarizedBlock()
		if pb == nil {
			Logger.Error("Found no block on previous round", zap.Int64("round", r))
			return errors.New("no previous round block")
		}

		blocks = append(blocks, pb)
		if r == lfr {
			Logger.Debug("Reached the latest finalized block round",
				zap.Int64("round", pb.Round),
				zap.Int64("current_round", round),
				zap.Int64("round_gap", round-pb.Round))
			break
		}
	}

	lastBlock := blocks[len(blocks)-1]
	if !lastBlock.IsStateComputed() {
		return errors.New("could not find block with computed state")
	}
	// calculate the state changes from the latest deterministic block
	for i := len(blocks) - 2; i >= 0; i-- {
		// get state change of the block
		blocks[i].CreateState(lastBlock.ClientState.GetNodeDB())
		c.GetBlockStateChange(blocks[i])
		blocks[i].SetPreviousBlock(blocks[i+1])
		lastBlock = blocks[i]
	}

	Logger.Debug("updateStateFromNetwork",
		zap.Int("num", len(blocks)-1),
		zap.Int64("start_round", lfr),
		zap.Int64("to_round", round))

	return nil
}

func (c *Chain) computeState(ctx context.Context, b *block.Block) error {
	if b.IsStateComputed() {
		return nil
	}
	pb := b.PrevBlock
	if pb == b {
		b.PrevBlock = nil // reset (a real case, may be unexpected)
	}
	if pb == nil {
		pb = c.GetPreviousBlock(ctx, b)
		if pb == nil {
			b.SetStateStatus(block.StateFailed)
			Logger.Error("compute state - previous block not available",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("prev_block", b.PrevHash))
			return ErrPreviousBlockUnavailable
		}
	}
	if pb == b || pb.StateMutex == b.StateMutex {
		b.PrevBlock = nil // reset (a real case, may be unexpected)
		Logger.Error("computing block state", zap.String("error",
			"block_prev points to itself, or its state mutex does it"))
		return common.NewError("computing block state",
			"prev_block points to itself, or its state mutex does it")
	}
	if !pb.IsStateComputed() {
		if pb.GetStateStatus() == block.StateFailed {
			if err := c.updateStateFromNetwork(ctx, pb); err != nil {
				Logger.Error("fetchMissingStates failed", zap.Error(err))
				return err
			}
			if !pb.IsStateComputed() {
				return ErrPreviousStateUnavailable
			}
		} else {
			Logger.Info("compute state - previous block state not ready",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.String("prev_block", b.PrevHash),
				zap.Int8("prev_block_state", pb.GetBlockState()),
				zap.Int8("prev_block_state_status", pb.GetStateStatus()))
			err := c.ComputeState(ctx, pb)
			if err != nil {
				pb.SetStateStatus(block.StateFailed)
				if state.DebugBlock() {
					Logger.Error("compute state - error computing previous state",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.String("prev_block", b.PrevHash), zap.Error(err))
				} else {
					Logger.Error("compute state - error computing previous state",
						zap.Int64("round", b.Round),
						zap.String("block", b.Hash),
						zap.String("prev_block", b.PrevHash), zap.Error(err))
				}
				return err
			}
		}
	}
	if pb.ClientState == nil {
		Logger.Error("compute state - previous state nil",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.String("prev_block", b.PrevHash),
			zap.Int8("prev_block_status", b.PrevBlock.GetStateStatus()))
		return ErrPreviousStateUnavailable
	}

	// Before continue the the following state update for transactions, the previous
	// block's state must be computed successfully.
	if !pb.IsStateComputed() {
		Logger.Error("previous state not compute successfully",
			zap.Int64("round", b.Round),
			zap.String("block", b.Hash),
			zap.Any("state status", pb.GetStateStatus()))
		return ErrPreviousStateNotComputed
	}
	b.SetStateDB(pb)

	Logger.Info("compute state", zap.Int64("round", b.Round),
		zap.String("block", b.Hash),
		zap.String("client_state", util.ToHex(b.ClientStateHash)),
		zap.String("begin_client_state", util.ToHex(b.ClientState.GetRoot())),
		zap.String("prev_block", b.PrevHash),
		zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)))

	for _, txn := range b.Txns {
		if datastore.IsEmpty(txn.ClientID) {
			txn.ComputeClientID()
		}
		if err := c.UpdateState(b, txn); err != nil {
			b.SetStateStatus(block.StateFailed)
			Logger.Error("compute state - update state failed",
				zap.Int64("round", b.Round),
				zap.String("block", b.Hash),
				zap.String("client_state", util.ToHex(b.ClientStateHash)),
				zap.String("prev_block", b.PrevHash),
				zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)),
				zap.Error(err))
			return common.NewError("state_update_error", "error updating state")
		}
	}
	if bytes.Compare(b.ClientStateHash, b.ClientState.GetRoot()) != 0 {
		b.SetStateStatus(block.StateFailed)
		Logger.Error("compute state - state hash mismatch",
			zap.Int64("round", b.Round), zap.String("block", b.Hash),
			zap.Int("block_size", len(b.Txns)),
			zap.Int("changes", len(b.ClientState.GetChangeCollector().GetChanges())),
			zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
			zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
		return ErrStateMismatch
	}
	c.StateSanityCheck(ctx, b)
	b.SetStateStatus(block.StateSuccessful)
	Logger.Info("compute state successful", zap.Int64("round", b.Round),
		zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)),
		zap.Int("changes", len(b.ClientState.GetChangeCollector().GetChanges())),
		zap.String("block_state_hash", util.ToHex(b.ClientStateHash)),
		zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
	return nil
}

//SaveChanges - persist the state changes
func (c *Chain) SaveChanges(ctx context.Context, b *block.Block) error {
	if !b.IsStateComputed() {
		err := errors.New("block state not computed")
		Logger.Error("save changes failed", zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return err
	}

	if b.ClientState == nil {
		Logger.Error("save changes - client state is nil",
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return errors.New("save changes - client state is nil")
	}

	lock := b.StateMutex
	lock.Lock()
	defer lock.Unlock()
	var err error
	ts := time.Now()
	switch b.GetStateStatus() {
	case block.StateSynched, block.StateSuccessful:
		err = b.ClientState.SaveChanges(c.stateDB, false)
		lndb, ok := b.ClientState.GetNodeDB().(*util.LevelNodeDB)
		if ok {
			c.stateDB.(*util.PNodeDB).TrackDBVersion(lndb.GetDBVersion())
		}
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
		Logger.Info("save state - slow", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(changes)), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration), zap.Duration("p95", time.Duration(math.Round(p95/1000000))*time.Millisecond))
	} else {
		Logger.Debug("save state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(changes)), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration))
	}
	if err != nil {
		Logger.Info("save state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(changes)), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.Duration("duration", duration), zap.Error(err))
	}

	return err
}

func (c *Chain) rebaseState(lfb *block.Block) {
	if lfb.ClientState == nil {
		return
	}
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	ndb := lfb.ClientState.GetNodeDB()
	if ndb != c.stateDB {
		Logger.Debug("finalize round - rebasing current state db",
			zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
			zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
		lfb.ClientState.SetNodeDB(c.stateDB)
		if lndb, ok := ndb.(*util.LevelNodeDB); ok {
			Logger.Debug("finalize round - rebasing current state db",
				zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
				zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
			lndb.RebaseCurrentDB(c.stateDB)
			Logger.Debug("finalize round - rebased current state db",
				zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
				zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
		}
		Logger.Debug("finalize round - rebased current state db",
			zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
			zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
	}
}

//ExecuteSmartContract - executes the smart contract for the transaction
func (c *Chain) ExecuteSmartContract(t *transaction.Transaction, balances bcstate.StateContextI) (string, error) {
	var output string
	var err error
	ts := time.Now()
	if balances.GetBlock().IsBlockNotarized() || c.SmartContractTimeout == 0 {
		output, err = smartcontract.ExecuteSmartContract(common.GetRootContext(), t, balances)
		SmartContractExecutionTimer.Update(time.Since(ts))
		return output, err
	}
	done := make(chan bool, 1)
	ctx, cancelf := context.WithTimeout(common.GetRootContext(), c.SmartContractTimeout)
	defer cancelf()
	go func() {
		output, err = smartcontract.ExecuteSmartContract(ctx, t, balances)
		done <- true
	}()
	select {
	case <-time.After(c.SmartContractTimeout):
		return "", common.NewError("smart_contract_execution_timeout", "smart contract execution timed out")
	case <-done:
		SmartContractExecutionTimer.Update(time.Since(ts))
		return output, err
	}
}

// UpdateState - update the state of the transaction w.r.t the given block.
// Note, don't call this from within state computation logic since their is
// already a lock on StateMutex. This API is for someone reading the state from
// outside the protocol without already holding a lock on StateMutex. The block
// starts off with the state from the prior block and as transactions are
// processed into a block, the state gets updated. If a state can't be updated
// (e.g low balance), then a false is returned so that the transaction will not
// make it into the block.
func (c *Chain) UpdateState(b *block.Block, txn *transaction.Transaction) error {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return c.updateState(b, txn)
}

// newStateContext creation helper.
func (c *Chain) newStateContext(b *block.Block, s util.MerklePatriciaTrieI,
	txn *transaction.Transaction) (balances *bcstate.StateContext) {

	return bcstate.NewStateContext(b, s, c.clientStateDeserializer, txn,
		c.GetBlockSharders, c.GetLatestFinalizedMagicBlock,
		c.GetSignatureScheme)
}

func (c *Chain) updateState(b *block.Block, txn *transaction.Transaction) (
	err error) {

	var (
		clientState = CreateTxnMPT(b.ClientState) // begin transaction
		startRoot   = clientState.GetRoot()
		sctx        = c.newStateContext(b, clientState, txn)
	)

	switch txn.TransactionType {

	case transaction.TxnTypeSmartContract:
		var output string
		if output, err = c.ExecuteSmartContract(txn, sctx); err != nil {
			Logger.Error("Error executing the SC", zap.Any("txn", txn),
				zap.Error(err))
			return
		}
		txn.TransactionOutput = output
		Logger.Info("SC executed with output",
			zap.Any("txn_output", txn.TransactionOutput),
			zap.Any("txn_hash", txn.Hash))

	case transaction.TxnTypeData:

	case transaction.TxnTypeSend:
		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, txn.ToClientID,
			state.Balance(txn.Value)))
		if err != nil {
			return
		}
	default:
		Logger.Error("Invalid transaction type", zap.Int("txn type", txn.TransactionType))
		return fmt.Errorf("invalid transaction type: %v", txn.TransactionType)
	}

	if config.DevConfiguration.IsFeeEnabled {
		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, minersc.ADDRESS,
			state.Balance(txn.Fee)))
		if err != nil {
			return
		}
	}

	if err = sctx.Validate(); err != nil {
		return
	}

	for _, transfer := range sctx.GetTransfers() {
		err = c.transferAmount(sctx, transfer.ClientID, transfer.ToClientID,
			state.Balance(transfer.Amount))
		if err != nil {
			return
		}
	}

	for _, signedTransfer := range sctx.GetSignedTransfers() {
		err = c.transferAmount(sctx, signedTransfer.ClientID,
			signedTransfer.ToClientID, state.Balance(signedTransfer.Amount))
		if err != nil {
			return
		}
	}

	for _, mint := range sctx.GetMints() {
		err = c.mintAmount(sctx, mint.ToClientID, state.Balance(mint.Amount))
		if err != nil {
			Logger.Error("mint error", zap.Any("error", err),
				zap.Any("transaction", txn.Hash))
			return
		}
	}

	// commit transaction
	if err = b.ClientState.MergeMPTChanges(clientState); err != nil {
		if state.DebugTxn() {
			Logger.DPanic("update state - merge mpt error",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Error(err))
		}

		Logger.Error("error committing txn", zap.Any("error", err))
		return
	}

	if state.DebugTxn() {
		if err = c.validateState(context.TODO(), b, startRoot); err != nil {
			Logger.DPanic("update state - state validation failure",
				zap.Any("txn", txn), zap.Error(err))
		}
		var os *state.State
		os, err = c.getState(b.ClientState, c.OwnerID)
		if err != nil || os == nil || os.Balance == 0 {
			Logger.DPanic("update state - owner account",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Any("os", os), zap.Error(err))
		}
	}

	txn.Status = transaction.TxnSuccess
	return
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
	if fromClient == toClient {
		return common.InvalidRequest("from and to client should be different for balance transfer")
	}
	b := sctx.GetBlock()
	clientState := sctx.GetState()
	txn := sctx.GetTransaction()
	fs, err := c.getState(clientState, fromClient)
	if !isValid(err) {
		if state.DebugTxn() {
			Logger.Error("transfer amount - client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
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
		if state.DebugTxn() {
			Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
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
	sctx.SetStateContext(fs)
	fs.Balance -= amount
	if fs.Balance == 0 {
		Logger.Info("transfer amount - remove client", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client", fromClient), zap.Any("txn", txn))
		_, err = clientState.Delete(util.Path(fromClient))
	} else {
		_, err = clientState.Insert(util.Path(fromClient), fs)
	}
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		return err
	}
	sctx.SetStateContext(ts)
	ts.Balance += amount
	_, err = clientState.Insert(util.Path(toClient), ts)
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		return err
	}
	return nil
}

func (c *Chain) mintAmount(sctx bcstate.StateContextI, toClient datastore.Key, amount state.Balance) error {
	if amount == 0 {
		return nil
	}
	b := sctx.GetBlock()
	clientState := sctx.GetState()
	txn := sctx.GetTransaction()
	ts, err := c.getState(clientState, toClient)
	if !isValid(err) {
		if state.DebugTxn() {
			Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
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
		if state.Debug() {
			Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		return err
	}
	sctx.SetStateContext(ts)
	ts.Balance += amount
	_, err = clientState.Insert(util.Path(toClient), ts)
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
				for _, txn := range b.Txns {
					if txn == nil {
						break
					}
					fmt.Fprintf(stateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
				}
				fmt.Fprintf(stateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
				printStates(clientState, b.ClientState)
				Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		if state.Debug() {
			Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		return err
	}
	return nil
}

func CreateTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion())
	tmpt.SetRoot(mpt.GetRoot())
	return tmpt
}

func (c *Chain) getState(clientState util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	if clientState == nil {
		return nil, common.NewError("getState", "client state does not exist")
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

/*GetState - Get the state of a client w.r.t a block. Note, don't call this from within state computation logic
since block.GetStateValue uses a RLock on the StateMutex. This API is for someone reading the state from outside
the protocol without already holding a lock on StateMutex */
func (c *Chain) GetState(b *block.Block, clientID string) (*state.State, error) {
	c.stateMutex.RLock()
	defer c.stateMutex.RUnlock()
	ss, err := b.ClientState.GetNodeValue(util.Path(clientID))
	if err != nil {
		if !b.IsStateComputed() {
			return nil, common.NewError("state_not_yet_computed", "State is not yet computed")
		}
		ps := c.GetPruneStats()
		if ps != nil && ps.MissingNodes > 0 {
			return nil, common.NewError("state_not_synched", "State sync is not yet complete")
		}
		return nil, err
	}
	st := c.clientStateDeserializer.Deserialize(ss).(*state.State)
	return st, nil
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
