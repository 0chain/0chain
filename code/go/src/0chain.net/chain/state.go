package chain

import (
	"bytes"
	"context"
	"fmt"

	"0chain.net/smartcontract"
	"0chain.net/smartcontractstate"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/config"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/state"
	"0chain.net/transaction"
	"0chain.net/util"
	"go.uber.org/zap"
)

var ErrPreviousStateUnavailable = common.NewError("prev_state_unavailable", "Previous state not available")

//StateMismatch - indicate if there is a mismatch between computed state and received state of a block
const StateMismatch = "state_mismatch"

var ErrStateMismatch = common.NewError(StateMismatch, "computed state hash doesn't match with the state hash of the block")

var ErrInsufficientBalance = common.NewError("insufficient_balance", "balance not sufficient for transfer")

/*ComputeState - compute the state for the block */
func (c *Chain) ComputeState(ctx context.Context, b *block.Block) error {
	lock := b.StateMutex
	if lock == nil {
		return common.NewError("invalid_block", "Invalid block")
	}
	lock.Lock()
	defer lock.Unlock()
	return c.computeState(ctx, b)
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
					Logger.Info("compute state - previous block not available", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
				}
			}
			return ErrPreviousBlockUnavailable
		}
	}
	if !pb.IsStateComputed() {
		Logger.Info("compute state - previous block state not ready", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Int8("prev_block_state", pb.GetBlockState()), zap.Int8("prev_block_state_status", pb.GetStateStatus()))
		err := c.ComputeState(ctx, pb)
		if err != nil {
			pb.SetStateStatus(block.StateFailed)
			if state.DebugBlock() {
				Logger.Error("compute state - error computing previous state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
			} else {
				if config.DevConfiguration.State {
					Logger.Info("compute state - error computing previous state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Error(err))
				}
			}
			return err
		}
	}
	/*
		if pb.ClientState == nil {
			if config.DevConfiguration.State {
				Logger.Error("compute state - previous state nil", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash))
			}
			return ErrPreviousStateUnavailable
		}*/
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

func (c *Chain) rebaseState(lfb *block.Block) {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	ndb := lfb.ClientState.GetNodeDB()
	if ndb != c.stateDB {
		lfb.ClientState.SetNodeDB(c.stateDB)
		if lndb, ok := ndb.(*util.LevelNodeDB); ok {
			Logger.Debug("finalize round - rebasing current state db", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash), zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
			lndb.RebaseCurrentDB(c.stateDB)
			lfb.ClientState.ResetChangeCollector(nil)
			Logger.Debug("finalize round - rebased current state db", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash), zap.String("hash", util.ToHex(lfb.ClientState.GetRoot())))
		}
	}

	sndb := lfb.SCStateDB
	if sndb != c.scStateDB {
		lfb.SCStateDB = c.scStateDB
		if lndb, ok := sndb.(*smartcontractstate.PipedSCDB); ok {
			Logger.Debug("finalize round - rebasing current smart contract state db", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash))
			lndb.RebaseCurrentDB(c.scStateDB)
			Logger.Debug("finalize round - rebased current smart contract state db", zap.Int64("round", lfb.Round), zap.String("block", lfb.Hash))
		}
	}
}

//ExecuteSmartContract - executes the smart contract for the transaction
func (c *Chain) ExecuteSmartContract(t *transaction.Transaction, ndb smartcontractstate.SCDB) (string, error) {
	output, err := smartcontract.ExecuteSmartContract(common.GetRootContext(), t, ndb)
	return output, err
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
	sctx := NewStateContext(b, clientState, c.clientStateDeserializer, txn)

	switch txn.TransactionType {
	case transaction.TxnTypeSmartContract:
		mndb := smartcontractstate.NewMemorySCDB()
		ndb := smartcontractstate.NewPipedSCDB(mndb, b.SCStateDB, false)
		output, err := c.ExecuteSmartContract(txn, ndb)
		if err != nil {
			Logger.Error("Smart contract execution returned error", zap.Any("error", err), zap.Any("transaction", txn.Hash))
			return false
		}
		err = smartcontractstate.SaveChanges(common.GetRootContext(), mndb, b.SCStateDB)
		if err != nil {
			Logger.Error("Error in saving the state on the block after execution", zap.Any("error", err))
			return false
		}
		txn.TransactionOutput = output
		txn.OutputHash = txn.ComputeOutputHash()
		Logger.Info("SC executed for transaction: ", zap.String("txn", txn.Hash), zap.String("output_hash", txn.OutputHash), zap.String("txn_output", txn.TransactionOutput))
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

	err := mergeMPT(b.ClientState, clientState) // commit transaction
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
func (c *Chain) transferAmount(sctx StateContextI, fromClient, toClient datastore.Key, amount state.Balance) error {
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
		if state.Debug() {
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(stateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(stateOut, "transfer amount - error getting state value: %v %+v %v\n", txn.ClientID, txn, err)
			printStates(clientState, b.ClientState)
			Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", txn.ClientID, err))
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
		if state.Debug() {
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(stateOut, "transfer amount r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(stateOut, "transfer amount - error getting state value: %v %+v %v\n", txn.ToClientID, txn, err)
			printStates(clientState, b.ClientState)
			Logger.DPanic(fmt.Sprintf("transfer amount - error getting state value: %v %v", txn.ToClientID, err))
		}
		return err
	}
	fs.SetRound(b.Round)
	fs.Balance -= amount
	if fs.Balance == 0 {
		Logger.Info("transfer amount - remove client", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client", txn.ClientID), zap.Any("txn", txn))
		_, err = clientState.Delete(util.Path(txn.ClientID))
	} else {
		_, err = clientState.Insert(util.Path(txn.ClientID), fs)
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
	_, err = clientState.Insert(util.Path(txn.ToClientID), ts)
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

func mergeMPT(mpt util.MerklePatriciaTrieI, mpt2 util.MerklePatriciaTrieI) error {
	if state.DebugTxn() {
		Logger.Debug("merge mpt", zap.String("mpt_root", util.ToHex(mpt.GetRoot())), zap.String("mpt2_root", util.ToHex(mpt2.GetRoot())))
	}
	return mpt.MergeMPT(mpt2)
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
