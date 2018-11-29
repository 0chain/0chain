package chain

import (
	"bytes"
	"context"
	"fmt"
	"os"

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
}

/*UpdateState - update the state of the transaction w.r.t the given block
* The block starts off with the state from the prior block and as transactions are processed into a block, the state gets updated
* If a state can't be updated (e.g low balance), then a false is returned so that the transaction will not make it into the block
 */
func (c *Chain) UpdateState(b *block.Block, txn *transaction.Transaction) bool {
	c.stateMutex.Lock()
	defer c.stateMutex.Unlock()
	clientState := createTxnMPT(b.ClientState)
	startRoot := clientState.GetRoot()
	fs, err := c.getState(clientState, txn.ClientID)
	if !isValid(err) {
		if config.DevConfiguration.State {
			Logger.Error("update state - client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
		}
		if state.Debug() {
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(stateOut, "update state r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
			}
			fmt.Fprintf(stateOut, "update state - error getting state value: %v %+v %v\n", txn.ClientID, txn, err)
			printStates(clientState, b.ClientState)
			Logger.DPanic(fmt.Sprintf("update state - error getting state value: %v %v", txn.ClientID, err))
		}
		return false
	}
	tbalance := state.Balance(txn.Value)
	switch txn.TransactionType {
	case transaction.TxnTypeData:
	case transaction.TxnTypeSend:
		if tbalance == 0 {
			return false
		}
		if fs.Balance < tbalance {
			return false
		}
		ts, err := c.getState(clientState, txn.ToClientID)
		if !isValid(err) {
			if config.DevConfiguration.State {
				Logger.Error("update state - to_client get", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			}
			if state.Debug() {
				for _, txn := range b.Txns {
					if txn == nil {
						break
					}
					fmt.Fprintf(stateOut, "update state r=%v b=%v t=%+v\n", b.Round, b.Hash, txn)
				}
				fmt.Fprintf(stateOut, "update state - error getting state value: %v %+v %v\n", txn.ToClientID, txn, err)
				printStates(clientState, b.ClientState)
				Logger.DPanic(fmt.Sprintf("update state - error getting state value: %v %v", txn.ToClientID, err))
			}
			return false
		}
		fs.SetRound(b.Round)
		fs.Balance -= tbalance
		if fs.Balance == 0 {
			Logger.Info("update state - remove client", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client", txn.ClientID), zap.Any("txn", txn))
			_, err = clientState.Delete(util.Path(txn.ClientID))
		} else {
			_, err = clientState.Insert(util.Path(txn.ClientID), fs)
		}
		if err != nil {
			if config.DevConfiguration.State {
				Logger.DPanic("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				Logger.Error("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
		ts.SetRound(b.Round)
		ts.Balance += tbalance
		_, err = clientState.Insert(util.Path(txn.ToClientID), ts)
		if err != nil {
			if config.DevConfiguration.State {
				Logger.DPanic("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			if state.Debug() {
				Logger.Error("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
		}
	}
	err = mergeMPT(b.ClientState, clientState)
	if err != nil {
		Logger.DPanic("update state - merge mpt error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
	}
	if state.DebugTxn() {
		if err := c.ValidateState(context.TODO(), b, startRoot); err != nil {
			Logger.DPanic("update state - state validation failure", zap.Any("txn", txn), zap.Error(err))
		}
		os, err := c.getState(b.ClientState, c.OwnerID)
		if err != nil || os == nil || os.Balance == 0 {
			Logger.DPanic("update state - owner account", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Any("os", os), zap.Error(err))
		}
	}
	return true
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
	} else {
		s = c.clientStateDeserializer.Deserialize(ss).(*state.State)
	}
	return s, nil
}

/*GetState - Get the state of a client w.r.t a finalized block */
func (c *Chain) GetState(fb *block.Block, clientID string) (*state.State, error) {
	return c.getState(fb.ClientState, clientID)
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

var stateOut *os.File

/*SetupStateLogger - a separate logger for state to be able to debug state */
func SetupStateLogger(file string) {
	out, err := os.Create(file)
	if err != nil {
		panic(err)
	}
	stateOut = out
	fmt.Fprintf(stateOut, "starting state log ...\n")
}

//StateSanityCheck - after generating a block or verification of a block, this can be called to run some state sanity checks
func (c *Chain) StateSanityCheck(ctx context.Context, b *block.Block) {
	if !state.DebugBlock() {
		return
	}
	if err := c.ValidateState(ctx, b, b.PrevBlock.ClientState.GetRoot()); err != nil {
		Logger.DPanic("state sanity check - state change validation", zap.Error(err))
	}
	if err := c.ValidateStateChangesRoot(b); err != nil {
		Logger.DPanic("state sanity check - state changes root validation", zap.Error(err))
	}
}

//ValidateState - validates the state of a block
func (c *Chain) ValidateState(ctx context.Context, b *block.Block, priorRoot util.Key) error {
	if len(b.ClientState.GetChangeCollector().GetChanges()) > 0 {
		changes := block.NewBlockStateChange(b)
		stateRoot := changes.GetRoot()
		if stateRoot == nil {
			if stateOut != nil {
				b.ClientState.PrettyPrint(stateOut)
			}
			if state.DebugBlock() {
				Logger.DPanic("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			} else {
				Logger.Error("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if bytes.Compare(stateRoot.GetHashBytes(), b.ClientState.GetRoot()) != 0 {
			if stateOut != nil {
				b.ClientState.GetChangeCollector().PrintChanges(stateOut)
				b.ClientState.PrettyPrint(stateOut)
			}
			if state.DebugBlock() {
				Logger.DPanic("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.Int("changes", len(changes.Nodes)))
			} else {
				Logger.Error("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if priorRoot == nil {
			priorRoot = b.PrevBlock.ClientState.GetRoot()
		}
		err := changes.Validate(ctx)
		if err != nil {
			Logger.Error("validate state - changes validate failure", zap.Error(err))
			pstate := util.CloneMPT(b.ClientState)
			pstate.SetRoot(priorRoot)
			printStates(b.ClientState, pstate)
			return err
		}
		err = b.ClientState.Validate()
		if err != nil {
			Logger.Error("validate state - client state validate failure", zap.Error(err))
			pstate := util.CloneMPT(b.ClientState)
			pstate.SetRoot(priorRoot)
			printStates(b.ClientState, pstate)
			/*
				if state.Debug() && stateOut != nil {
					fmt.Fprintf(stateOut, "previous block\n")
					if bytes.Compare(b.PrevBlock.ClientState.GetRoot(), priorRoot) != 0 {
						b.PrevBlock.ClientState.PrettyPrint(stateOut)
					}
				}*/
			return err
		}
	}
	/*
		if b.Round > 15 {
			state.SetDebugLevel(state.DebugLevelTxn)
		}*/
	return nil
}

//ValidateStateChangesRoot - validates that root computed from changes matches with the state root
func (c *Chain) ValidateStateChangesRoot(b *block.Block) error {
	bsc := block.NewBlockStateChange(b)
	if b.ClientStateHash != nil && (bsc.GetRoot() == nil || bytes.Compare(bsc.GetRoot().GetHashBytes(), b.ClientStateHash) != 0) {
		computedRoot := ""
		if bsc.GetRoot() != nil {
			computedRoot = bsc.GetRoot().GetHash()
		}
		Logger.Error("new block state change - root mismatch", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("state_root", util.ToHex(b.ClientStateHash)), zap.Any("computed_root", computedRoot))
		return ErrStateMismatch
	}
	return nil
}

func printStates(cstate util.MerklePatriciaTrieI, pstate util.MerklePatriciaTrieI) {
	if !state.Debug() || stateOut == nil {
		return
	}
	cstate.PrettyPrint(stateOut)
	fmt.Fprintf(stateOut, "previous state\n")
	pstate.PrettyPrint(stateOut)
}
