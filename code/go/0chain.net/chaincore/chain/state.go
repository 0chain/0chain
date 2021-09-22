package chain

import (
	"context"
	"fmt"
	"time"

	"errors"

	"0chain.net/chaincore/block"
	bcstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/minersc"
	metrics "github.com/rcrowley/go-metrics"
	"go.uber.org/zap"
)

//SmartContractExecutionTimer - a metric that tracks the time it takes to execute a smart contract txn
var SmartContractExecutionTimer metrics.Timer

func init() {
	SmartContractExecutionTimer = metrics.GetOrRegisterTimer("sc_execute_timer", nil)
}

var ErrInsufficientBalance = common.NewError("insufficient_balance", "Balance not sufficient for transfer")

/*ComputeState - compute the state for the block */
func (c *Chain) ComputeState(ctx context.Context, b *block.Block) error {
	return c.computeState(ctx, b)
}

// ComputeOrSyncState - try to compute state and if there is an error, just sync it
func (c *Chain) ComputeOrSyncState(ctx context.Context, b *block.Block) error {
	err := c.computeState(ctx, b)
	if err != nil {
		bsc, err := c.getBlockStateChange(b)
		if err != nil {
			return err
		}
		if bsc != nil {
			if err = c.ApplyBlockStateChange(b, bsc); err != nil {
				logging.Logger.Error("compute state - applying state change",
					zap.Any("round", b.Round), zap.Any("block", b.Hash),
					zap.Error(err))
				return err
			}
		}
		if !b.IsStateComputed() {
			logging.Logger.Error("compute state - state change error",
				zap.Any("round", b.Round), zap.Any("block", b.Hash),
				zap.Error(err))
			return err
		}
	}
	return nil
}

func (c *Chain) computeState(ctx context.Context, b *block.Block) error {
	return b.ComputeState(ctx, c)
}

//SaveChanges - persist the state changes
func (c *Chain) SaveChanges(ctx context.Context, b *block.Block) error {
	if !b.IsStateComputed() {
		err := errors.New("block state not computed")
		logging.Logger.Error("save changes failed", zap.Error(err),
			zap.Int64("round", b.Round),
			zap.String("hash", b.Hash))
		return err
	}
	return b.SaveChanges(ctx, c)
}

func (c *Chain) rebaseState(lfb *block.Block) {
	if lfb.ClientState == nil {
		return
	}
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	ndb := lfb.ClientState.GetNodeDB()
	if ndb != c.stateDB {
		logging.Logger.Debug("finalize round - rebasing current state db",
			zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
			zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
		lfb.ClientState.SetNodeDB(c.stateDB)
		if lndb, ok := ndb.(*util.LevelNodeDB); ok {
			logging.Logger.Debug("finalize round - rebasing current state db",
				zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
				zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
			lndb.RebaseCurrentDB(c.stateDB)
			logging.Logger.Debug("finalize round - rebased current state db",
				zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
				zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
		}
		logging.Logger.Debug("finalize round - rebased current state db",
			zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash),
			zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
	}
}

//ExecuteSmartContract - executes the smart contract for the transaction
func (c *Chain) ExecuteSmartContract(ctx context.Context, t *transaction.Transaction, balances bcstate.StateContextI) (string, error) {
	var output string
	var err error
	ts := time.Now()
	done := make(chan bool, 1)
	cctx, cancelf := context.WithTimeout(ctx, c.SmartContractTimeout)
	defer cancelf()
	go func() {
		output, err = smartcontract.ExecuteSmartContract(cctx, t, balances)
		done <- true
	}()
	select {
	case <-cctx.Done():
		return "", common.NewError("smart_contract_execution_ctx_err", cctx.Err().Error())
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
func (c *Chain) UpdateState(ctx context.Context, b *block.Block, txn *transaction.Transaction) (rset, wset map[datastore.Key]bool, err error) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	return c.updateState(ctx, b, txn)
}

// NewStateContext creation helper.
func (c *Chain) NewStateContext(b *block.Block, s util.MerklePatriciaTrieI,
	txn *transaction.Transaction) (balances *bcstate.StateContext) {

	return bcstate.NewStateContext(b, s, c.clientStateDeserializer,
		txn,
		c.GetBlockSharders,
		c.GetLatestFinalizedMagicBlock,
		c.GetCurrentMagicBlock,
		c.GetSignatureScheme)
}

func (c *Chain) updateState(ctx context.Context, b *block.Block, txn *transaction.Transaction) (rset map[datastore.Key]bool, wset map[datastore.Key]bool, err error) {

	// check if the block's ClientState has root value
	_, err = b.ClientState.GetNodeDB().GetNode(b.ClientState.GetRoot())
	if err != nil {
		return nil, nil, common.NewErrorf("update_state_failed",
			"block state root is incorrect, block hash: %v, state hash: %v, root: %v, round: %d",
			b.Hash, b.ClientStateHash, b.ClientState.GetRoot(), b.Round)
	}

	var (
		clientState = CreateTxnMPT(b.ClientState) // begin transaction
		startRoot   = clientState.GetRoot()
		sctx        = c.NewStateContext(b, clientState, txn)
	)

	switch txn.TransactionType {

	case transaction.TxnTypeSmartContract:
		var output string
		t := time.Now()
		if output, err = c.ExecuteSmartContract(ctx, txn, sctx); err != nil {
			logging.Logger.Error("Error executing the SC", zap.Any("txn", txn),
				zap.Error(err))
			return
		}
		txn.TransactionOutput = output
		logging.Logger.Info("SC executed with output",
			zap.Any("txn_output", txn.TransactionOutput),
			zap.Any("txn_hash", txn.Hash),
			zap.Any("txn_exec_time", time.Since(t)))

	case transaction.TxnTypeData:

	case transaction.TxnTypeSend:
		err = sctx.AddTransfer(state.NewTransfer(txn.ClientID, txn.ToClientID,
			state.Balance(txn.Value)))
		if err != nil {
			return
		}
	default:
		logging.Logger.Error("Invalid transaction type", zap.Int("txn type", txn.TransactionType))
		return nil, nil, fmt.Errorf("invalid transaction type: %v", txn.TransactionType)
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
		err = c.transferAmount(sctx, transfer.ClientID, transfer.ToClientID, transfer.Amount)
		if err != nil {
			return
		}
	}

	for _, signedTransfer := range sctx.GetSignedTransfers() {
		err = c.transferAmount(sctx, signedTransfer.ClientID,
			signedTransfer.ToClientID, signedTransfer.Amount)
		if err != nil {
			return
		}
	}

	for _, mint := range sctx.GetMints() {
		err = c.mintAmount(sctx, mint.ToClientID, mint.Amount)
		if err != nil {
			logging.Logger.Error("mint error", zap.Any("error", err),
				zap.Any("transaction", txn.Hash))
			// Temporary disable returning on mint error: TODO: revert back @bbist
			// return
		}
	}

	// commit transaction
	if err = b.ClientState.MergeMPTChanges(clientState); err != nil {
		if state.DebugTxn() {
			logging.Logger.DPanic("update state - merge mpt error",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Error(err))
		}

		logging.Logger.Error("error committing txn", zap.Any("error", err))
		return
	}

	if state.DebugTxn() {
		if err = block.ValidateState(context.TODO(), b, startRoot); err != nil {
			logging.Logger.DPanic("update state - state validation failure",
				zap.Any("txn", txn), zap.Error(err))
		}
		var os *state.State
		os, err = c.getState(b.ClientState, c.OwnerID)
		if err != nil || os == nil || os.Balance == 0 {
			logging.Logger.DPanic("update state - owner account",
				zap.Int64("round", b.Round), zap.String("block", b.Hash),
				zap.Any("txn", txn), zap.Any("os", os), zap.Error(err))
		}
	}

	txn.Status = transaction.TxnSuccess
	rset, wset = sctx.GetRWSets()
	return rset, wset, nil
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
	txn := sctx.GetTransaction()
	fs, err := sctx.GetClientState(fromClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", fromClient, txn, err)
			sctx.PrintStates()
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", fromClient, err))
		}
		return err
	}
	if fs.Balance < amount {
		return ErrInsufficientBalance
	}
	ts, err := sctx.GetClientState(toClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
			sctx.PrintStates()
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", toClient, err))
		}
		return err
	}
	sctx.SetStateContext(fs)
	fs.Balance -= amount
	if fs.Balance == 0 {
		logging.Logger.Info("transfer amount - remove client", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client", fromClient), zap.Any("txn", txn))
		_, err = sctx.DeleteClientTrieNode(fromClient)
	} else {
		_, err = sctx.InsertClientTrieNode(fromClient, fs)
	}
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		return err
	}
	sctx.SetStateContext(ts)
	ts.Balance += amount
	_, err = sctx.InsertClientTrieNode(toClient, ts)
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
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
	txn := sctx.GetTransaction()
	ts, err := sctx.GetClientState(toClient)
	if !isValid(err) {
		if state.DebugTxn() {
			logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
			sctx.PrintStates()
			logging.Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", toClient, err))
		}
		if state.Debug() {
			logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		return err
	}
	sctx.SetStateContext(ts)
	ts.Balance += amount
	_, err = sctx.InsertClientTrieNode(toClient, ts)
	if err != nil {
		if state.DebugTxn() {
			if config.DevConfiguration.State {
				logging.Logger.Error("transfer amount - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
				for _, txn := range b.Txns {
					if txn == nil {
						break
					}
					fmt.Fprintf(block.StateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
				}
				fmt.Fprintf(block.StateOut, "transfer amount - error getting state value: %v %+v %v\n", toClient, txn, err)
				sctx.PrintStates()
				logging.Logger.DPanic("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		if state.Debug() {
			logging.Logger.Error("transfer amount - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		return err
	}
	return nil
}

func CreateTxnMPT(mpt util.MerklePatriciaTrieI) util.MerklePatriciaTrieI {
	tdb := util.NewLevelNodeDB(util.NewMemoryNodeDB(), mpt.GetNodeDB(), false)
	tmpt := util.NewMerklePatriciaTrie(tdb, mpt.GetVersion(), mpt.GetRoot())
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
