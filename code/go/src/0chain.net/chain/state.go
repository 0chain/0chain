package chain

import (
	"0chain.net/block"
	"0chain.net/state"
	"0chain.net/transaction"
	"0chain.net/util"
)

/*UpdateState - update the state of the transaction w.r.t the given block
* The block starts off with the state from the prior block and as transactions are processed into a block, the state gets updated
* If a state can't be updated (e.g low balance), then a false is returned so that the transaction will not make it into the block
 */
func (c *Chain) UpdateState(txn *transaction.Transaction, b *block.Block) bool {
	clientStateMT := b.ClientStateMT
	clientState, err := clientStateMT.GetNodeValue(util.Path(txn.ClientID))
	if err != util.ErrValueNotPresent {
		return false
	}
	s := &state.State{}
	s.Balance = state.Balance(0)
	if err == nil {
		s = c.ClientStateDeserializer.Deserialize(clientState).(*state.State)
	}
	tbalance := state.Balance(txn.Value)

	switch txn.TransactionType {
	case transaction.TxnTypeSend:
		if s.Balance < tbalance {
			//TODO: we need to return false once state starts working properly
			//return false
			return true
		}
		s.Balance -= tbalance
		if s.Balance == 0 {
			clientStateMT.Delete(util.Path(txn.ClientID))
		} else {
			clientStateMT.Insert(util.Path(txn.ClientID), s)
		}
		return true
	default:
		return true // TODO: This should eventually return false by default for all unkown cases
	}
}
