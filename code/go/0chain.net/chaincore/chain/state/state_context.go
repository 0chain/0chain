package state

import (
	"context"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

const approvedMinter = "6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9"

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
	AddMint(m *state.Mint) error
	GetTransfers() []*state.Transfer
	GetMints() []*state.Mint
	Validate(ctx context.Context) error
}

//StateContext - a context object used to manipulate global state
type StateContext struct {
	block                   *block.Block
	state                   util.MerklePatriciaTrieI
	txn                     *transaction.Transaction
	transfers               []*state.Transfer
	signedTransfers         []*state.SignedTransfer
	mints                   []*state.Mint
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

//AddSignedTransfer - add the signed transfer
func (sc *StateContext) AddSignedTransfer(st *state.SignedTransfer) {
	// Signature on the signed transfer will be checked on call to sc.Validate()
	sc.signedTransfers = append(sc.signedTransfers, st)
}

//AddMint - add the mint
func (sc *StateContext) AddMint(m *state.Mint) error {
	if m.Minter != approvedMinter || sc.txn.ToClientID != approvedMinter {
		return state.ErrInvalidMint
	}
	sc.mints = append(sc.mints, m)
	return nil
}

//GetTransfers - get all the transfers
func (sc *StateContext) GetTransfers() []*state.Transfer {
	return sc.transfers
}

//GetTransfers - get all the transfers
func (sc *StateContext) GetSignedTransfers() []*state.SignedTransfer {
	return sc.signedTransfers
}

//GetMints - get all the mints and fight bad breath
func (sc *StateContext) GetMints() []*state.Mint {
	return sc.mints
}

//Validate - implement interface
func (sc *StateContext) Validate(ctx context.Context) error {
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

	for _, signedTransfer := range sc.signedTransfers {
		err := signedTransfer.VerifySignature(ctx)
		if err != nil {
			return err
		}
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
