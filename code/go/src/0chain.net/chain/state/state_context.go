package state

import (
	"0chain.net/block"
	"0chain.net/datastore"
	"0chain.net/state"
	"0chain.net/transaction"
	"0chain.net/util"
)

/*
* The state context is available to the smart contract logic.
* The smart contract logic can use
*    GetClientBalance - to get the balance of a client at the beginning of executing the transaction.
*    AddTransfer - to add transfer of tokens from one client to another.
*  Restrictions:
*    1) The total transfer out from the txn.ClientID should be <= txn.Value
*    2) The only from clients valid are txn.ClientID and txn.ToClientID (which will be the smart contract's client id)
 */

//StateContextI - a state context interface. These interface are available for the smart contract
type StateContextI interface {
	GetBlock() *block.Block
	GetState() util.MerklePatriciaTrieI
	GetTransaction() *transaction.Transaction
	GetClientBalance(clientID datastore.Key) (state.Balance, error)
	AddTransfer(t *state.Transfer) error
	GetTransfers() []*state.Transfer
	Validate() error
}

//StateContext - a context object used to manipulate global state
type StateContext struct {
	block                   *block.Block
	state                   util.MerklePatriciaTrieI
	txn                     *transaction.Transaction
	transfers               []*state.Transfer
	clientStateDeserializer state.DeserializerI
}

//NewStateContext - create a new state context
func NewStateContext(b *block.Block, s util.MerklePatriciaTrieI, csd state.DeserializerI, t *transaction.Transaction) *StateContext {
	ctx := &StateContext{block: b, state: s, clientStateDeserializer: csd, txn: t}
	return ctx
}

//GetBlock - get the block associated with this state context
func (sc *StateContext) GetBlock() *block.Block {
	return sc.block
}

//GetState - get the state MPT associated with this state context
func (sc *StateContext) GetState() util.MerklePatriciaTrieI {
	return sc.state
}

//GetTransaction - get the transaction associated with this context
func (sc *StateContext) GetTransaction() *transaction.Transaction {
	return sc.txn
}

//AddTransfer - add the transfer
func (sc *StateContext) AddTransfer(t *state.Transfer) error {
	if t.ClientID != sc.txn.ClientID && t.ClientID != sc.txn.ToClientID {
		return state.ErrInvalidTransfer
	}
	sc.transfers = append(sc.transfers, t)
	return nil
}

//GetTransfers - get all the transfers
func (sc *StateContext) GetTransfers() []*state.Transfer {
	return sc.transfers
}

//Validate - implement interface
func (sc *StateContext) Validate() error {
	var amount state.Balance
	for _, transfer := range sc.transfers {
		if transfer.ClientID == sc.txn.ClientID {
			amount += transfer.Amount
		} else {
			if transfer.ClientID != sc.txn.ToClientID {
				return state.ErrInvalidTransfer
			}
		}
	}
	if amount > state.Balance(sc.txn.Value) {
		return state.ErrInvalidTransfer
	}
	return nil
}

func (sc *StateContext) getClientState(clientID string) (*state.State, error) {
	s := &state.State{}
	s.Balance = state.Balance(0)
	ss, err := sc.state.GetNodeValue(util.Path(clientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return s, err
	}
	s = sc.clientStateDeserializer.Deserialize(ss).(*state.State)
	//TODO: should we apply the pending transfers?
	return s, nil
}

//GetClientBalance - get the balance of the client
func (sc *StateContext) GetClientBalance(clientID string) (state.Balance, error) {
	s, err := sc.getClientState(clientID)
	if err != nil {
		return 0, err
	}
	return s.Balance, nil
}
