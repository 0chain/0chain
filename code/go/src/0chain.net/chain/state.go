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
			if state.DebugState {
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
			if state.DebugState {
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
				if config.DevConfiguration.State {
					Logger.Error("compute state - update state failed", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("client_state", util.ToHex(b.ClientStateHash)), zap.String("prev_block", b.PrevHash), zap.String("prev_client_state", util.ToHex(pb.ClientStateHash)))
				}
				return common.NewError("state_update_error", "error updating state")
			}
		}
	}
	if bytes.Compare(b.ClientStateHash, b.ClientState.GetRoot()) != 0 {
		b.SetStateStatus(block.StateFailed)
		if config.DevConfiguration.State {
			Logger.Error("compute state - state hash mismatch", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(b.ClientState.GetChangeCollector().GetChanges())), zap.String("block_state_hash", util.ToHex(b.ClientStateHash)), zap.String("computed_state_hash", util.ToHex(b.ClientState.GetRoot())))
		}
		return common.NewError(StateMismatch, "computed state hash doesn't match with the state hash of the block")
	}
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
	clientState := b.ClientState
	startRoot := clientState.GetRoot()
	prevRoot := startRoot
	fs, err := c.getState(clientState, txn.ClientID)
	if !isValid(err) {
		if b.Hash != "" || config.DevConfiguration.State {
			prevState := ""
			if b.PrevBlock != nil {
				prevState = util.ToHex(b.PrevBlock.ClientState.GetRoot())
			}
			Logger.Debug("update state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.String("prev_client_state", prevState), zap.Any("txn", txn), zap.Error(err))
		}

		if config.DevConfiguration.State && stateOut != nil {
			for _, txn := range b.Txns {
				if txn == nil {
					break
				}
				fmt.Fprintf(stateOut, "update state r=%v b=%v t=%v\n", b.Round, b.Hash, txn)
			}
			clientState.PrettyPrint(os.Stdout)
			Logger.DPanic(fmt.Sprintf("error getting state value: %v %v", txn.ClientID, err))
		}
		return false
	}
	tbalance := state.Balance(txn.Value)
	switch txn.TransactionType {
	case transaction.TxnTypeData:
	case transaction.TxnTypeSend:
		if fs.Balance < tbalance {
			return false
		}
		ts, err := c.getState(clientState, txn.ToClientID)
		if !isValid(err) {
			if config.DevConfiguration.State {
				Logger.Error("update state (to client)", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.String("prev_block", b.PrevHash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
				for _, txn := range b.Txns {
					if txn != nil {
						Logger.Info("update state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn))
					}
				}
				clientState.PrettyPrint(os.Stdout)
				Logger.DPanic(fmt.Sprintf("error getting state value: %v %v", txn.ToClientID, err))
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
			if state.DebugState {
				Logger.DPanic("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			Logger.Error("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		if state.DebugState {
			if err := c.ValidateState(context.TODO(), b); err != nil {
				clientState.SetRoot(prevRoot)
				if stateOut != nil {
					fmt.Fprintf(stateOut, "prior state\n")
					clientState.PrettyPrint(stateOut)
					stateOut.Sync()
				}
				Logger.DPanic("state validation failure", zap.Any("txn", txn), zap.Error(err))
			}
		}
		prevRoot = clientState.GetRoot()
		ts.SetRound(b.Round)
		ts.Balance += tbalance
		_, err = clientState.Insert(util.Path(txn.ToClientID), ts)
		if err != nil {
			if state.DebugState {
				Logger.DPanic("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
			}
			Logger.Error("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Error(err))
		}
		if state.DebugState {
			if err := c.ValidateState(context.TODO(), b); err != nil {
				clientState.SetRoot(prevRoot)
				if stateOut != nil {
					fmt.Fprintf(stateOut, "prior state\n")
					clientState.PrettyPrint(stateOut)
					stateOut.Sync()
				}
				Logger.DPanic("state validation failure", zap.Any("txn", txn), zap.Error(err))
			}
		}
	}
	if state.DebugState {
		os, err := c.getState(clientState, c.OwnerID)
		if err != nil || os == nil || os.Balance == 0 {
			Logger.DPanic("update state - owner account", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", txn), zap.Any("os", os), zap.Error(err))
		}
	}
	return true
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

//ValidateState - validates the state of a block
func (c *Chain) ValidateState(ctx context.Context, b *block.Block) error {
	if len(b.ClientState.GetChangeCollector().GetChanges()) > 0 {
		changes := block.NewBlockStateChange(b)
		stateRoot := changes.GetRoot()
		if stateRoot == nil {
			if stateOut != nil {
				b.ClientState.PrettyPrint(stateOut)
			}
			if state.DebugState {
				Logger.DPanic("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			} else {
				Logger.Error("validate state - state root is null", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("changes", len(changes.Nodes)))
			}
		}
		if bytes.Compare(stateRoot.GetHashBytes(), b.ClientState.GetRoot()) != 0 {
			root := changes.ComputeRootDebug()
			if stateOut != nil {
				b.ClientState.GetChangeCollector().PrintChanges(stateOut)
				b.ClientState.PrettyPrint(stateOut)
			}
			if state.DebugState {
				Logger.DPanic("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.String("computed_root", root.GetHash()), zap.Int("changes", len(changes.Nodes)))
			} else {
				Logger.Error("validate state", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("state", util.ToHex(b.ClientState.GetRoot())), zap.String("computed_state", stateRoot.GetHash()), zap.String("computed_root", root.GetHash()), zap.Int("changes", len(changes.Nodes)))
			}
		}
		err := changes.Validate(ctx)
		if err != nil {
			if state.DebugState && stateOut != nil {
				b.ClientState.PrettyPrint(stateOut)
			}
		}
		return err
	}
	return nil
}
