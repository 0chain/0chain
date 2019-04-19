package state

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/util"
)

var (
	approvedMinters = []string{
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d9",
		"6dba10422e368813802877a85039d3985d96760ed844092319743fb3a76712d1"}
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
	SetStateContext(st *state.State) error
	GetTrieNode(key datastore.Key) (util.Serializable, error)
	InsertTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error)
	DeleteTrieNode(key datastore.Key) (datastore.Key, error)
	AddTransfer(t *state.Transfer) error
	AddMint(m *state.Mint) error
	GetTransfers() []*state.Transfer
	GetMints() []*state.Mint
	Validate() error
	GetBlockSharders(b *block.Block) []string
}

//StateContext - a context object used to manipulate global state
type StateContext struct {
	block                   *block.Block
	state                   util.MerklePatriciaTrieI
	txn                     *transaction.Transaction
	transfers               []*state.Transfer
	mints                   []*state.Mint
	clientStateDeserializer state.DeserializerI
	getSharders             func(*block.Block) []string
}

//NewStateContext - create a new state context
func NewStateContext(b *block.Block, s util.MerklePatriciaTrieI, csd state.DeserializerI, t *transaction.Transaction, getSharderFunc func(*block.Block) []string) *StateContext {
	ctx := &StateContext{block: b, state: s, clientStateDeserializer: csd, txn: t, getSharders: getSharderFunc}
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

//AddMint - add the mint
func (sc *StateContext) AddMint(m *state.Mint) error {
	if !sc.isApprovedMinter(m) {
		return state.ErrInvalidMint
	}
	sc.mints = append(sc.mints, m)
	return nil
}

func (sc *StateContext) isApprovedMinter(m *state.Mint) bool {
	for _, minter := range approvedMinters {
		if m.Minter == minter && sc.txn.ToClientID == minter {
			return true
		}
	}
	return false
}

//GetTransfers - get all the transfers
func (sc *StateContext) GetTransfers() []*state.Transfer {
	return sc.transfers
}

//GetMints - get all the mints and fight bad breath
func (sc *StateContext) GetMints() []*state.Mint {
	return sc.mints
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
	totalValue := state.Balance(sc.txn.Value)
	if config.DevConfiguration.IsFeeEnabled {
		totalValue += state.Balance(sc.txn.Fee)
	}
	if amount > totalValue {
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

func (sc *StateContext) GetBlockSharders(b *block.Block) []string {
	return sc.getSharders(b)
}

func (sc *StateContext) GetTrieNode(key datastore.Key) (util.Serializable, error) {
	if encryption.IsHash(key) {
		return nil, common.NewError("failed to get trie node", "key is too short")
	}
	return sc.state.GetNodeValue(util.Path(key))
}

func (sc *StateContext) InsertTrieNode(key datastore.Key, node util.Serializable) (datastore.Key, error) {
	if encryption.IsHash(key) {
		return "", common.NewError("failed to get trie node", "key is too short")
	}
	byteKey, err := sc.state.Insert(util.Path(key), node)
	return datastore.Key(byteKey), err
}

func (sc *StateContext) DeleteTrieNode(key datastore.Key) (datastore.Key, error) {
	if encryption.IsHash(key) {
		return "", common.NewError("failed to get trie node", "key is too short")
	}
	byteKey, err := sc.state.Delete(util.Path(key))
	return datastore.Key(byteKey), err
}

//SetStateContext - set the state context
func (sc *StateContext) SetStateContext(s *state.State) error {
	s.SetRound(sc.block.Round)
	return s.SetTxnHash(sc.txn.Hash)
}
