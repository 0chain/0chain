package chain

import (
	"bytes"
	"context"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"0chain.net/state"
	"0chain.net/transaction"
	"0chain.net/util"
	"go.uber.org/zap"
)

const StateMismatch = "state_mismatch"

/*ComputeState - compute the state for the block */
func (c *Chain) ComputeState(ctx context.Context, b *block.Block) error {
	for _, txn := range b.Txns {
		if !c.UpdateState(b, txn) {
			return common.NewError("state_update_error", "error updating state")
		}
	}
	if bytes.Compare(b.ClientStateHash, b.ClientState.GetChangeCollector().GetRoot()) != 0 {
		Logger.Error("validate transaction state hash error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Int("block_size", len(b.Txns)), zap.Int("changes", len(b.ClientState.GetChangeCollector().GetChanges())), zap.String("block_state_hash", util.ToHex(b.ClientStateHash)), zap.String("computed_state_hash", util.ToHex(b.ClientState.GetChangeCollector().GetRoot())))
		return common.NewError(StateMismatch, "computed state hash doesn't match with the state hash of the block")
	}
	return nil
}

/*UpdateState - update the state of the transaction w.r.t the given block
* The block starts off with the state from the prior block and as transactions are processed into a block, the state gets updated
* If a state can't be updated (e.g low balance), then a false is returned so that the transaction will not make it into the block
 */
func (c *Chain) UpdateState(b *block.Block, txn *transaction.Transaction) bool {
	clientState := b.ClientState
	fs, err := c.getState(clientState, txn.ClientID)
	if err != nil {
		Logger.Error("update state", zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
		return false
	}
	tbalance := state.Balance(txn.Value)
	switch txn.TransactionType {
	case transaction.TxnTypeSend:
		if fs.Balance < tbalance {
			//TODO: we need to return false once state starts working properly
			//return false
			return true
		}
		fs.Balance -= tbalance
		if fs.Balance == 0 {
			_, err = clientState.Delete(util.Path(txn.ClientID))
		} else {
			_, err = clientState.Insert(util.Path(txn.ClientID), fs)
		}
		clientState.Insert(util.Path(txn.ToClientID), fs)
		if err != nil {
			Logger.Error("update state - error", zap.Int64("round", b.Round), zap.String("block", b.Hash), zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
		}
		ts, err := c.getState(clientState, txn.ToClientID)
		if err != nil {
			Logger.Error("update state (to client)", zap.Any("txn", datastore.ToJSON(txn)), zap.Error(err))
			return false
		}
		ts.Balance += tbalance
		clientState.Insert(util.Path(txn.ToClientID), ts)
		return true
	default:
		return true // TODO: This should eventually return false by default for all unkown cases
	}
}

func (c *Chain) getState(clientState util.MerklePatriciaTrieI, clientID string) (*state.State, error) {
	s := &state.State{}
	s.Balance = state.Balance(0)
	ss, err := clientState.GetNodeValue(util.Path(clientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
	} else {
		s = c.ClientStateDeserializer.Deserialize(ss).(*state.State)
	}
	return s, nil
}
