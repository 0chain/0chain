package state

import (
	"0chain.net/block"
	"0chain.net/datastore"
	"0chain.net/transaction"
	"0chain.net/util"
)

//ContextI - a state context interface. These interface are available for the smart contract
//A smart contract can add
type ContextI interface {
	GetBlock() *block.Block
	GetState() util.MerklePatriciaTrieI
	GetTransaction() *transaction.Transaction
	GetClientBalance(clientID datastore.Key) (Balance, error)
	AddTransfer(t *Transfer) error
	GetTransfers() []*Transfer
	Validate() error
}

//Context - a context object used to manipulate global state
type Context struct {
	block                   *block.Block
	state                   util.MerklePatriciaTrieI
	txn                     *transaction.Transaction
	transfers               []*Transfer
	clientStateDeserializer DeserializerI
}

//NewStateContext - create a new state context
func NewStateContext(b *block.Block, s util.MerklePatriciaTrieI, csd DeserializerI, t *transaction.Transaction) *Context {
	ctx := &Context{block: b, state: s, clientStateDeserializer: csd, txn: t}
	return ctx
}

//GetBlock - get the block associated with this state context
func (c *Context) GetBlock() *block.Block {
	return c.block
}

//GetState - get the state MPT associated with this state context
func (c *Context) GetState() util.MerklePatriciaTrieI {
	return c.state
}

//GetTransaction - get the transaction associated with this context
func (c *Context) GetTransaction() *transaction.Transaction {
	return c.txn
}

//AddTransfer - add the transfer
func (c *Context) AddTransfer(t *Transfer) error {
	if t.ClientID != c.txn.ClientID && t.ClientID != c.txn.ToClientID {
		return ErrInvalidTransfer
	}
	c.transfers = append(c.transfers, t)
	return nil
}

//GetTransfers - get all the transfers
func (c *Context) GetTransfers() []*Transfer {
	return c.transfers
}

//Validate - implement interface
func (c *Context) Validate() error {
	var amount Balance
	for _, transfer := range c.transfers {
		if transfer.ClientID == c.txn.ClientID {
			amount += transfer.Amount
		} else {
			if transfer.ClientID != c.txn.ToClientID {
				return ErrInvalidTransfer
			}
		}
	}
	if amount > Balance(c.txn.Value) {
		return ErrInvalidTransfer
	}
	return nil
}

func (c *Context) getClientState(clientID string) (*State, error) {
	s := &State{}
	s.Balance = Balance(0)
	ss, err := c.state.GetNodeValue(util.Path(clientID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return nil, err
		}
		return s, err
	}
	s = c.clientStateDeserializer.Deserialize(ss).(*State)
	//TODO: should we apply the pending transfers?
	return s, nil
}

//GetClientBalance - get the balance of the client
func (c *Context) GetClientBalance(clientID string) (Balance, error) {
	s, err := c.getClientState(clientID)
	if err != nil {
		return 0, err
	}
	return s.Balance, nil
}
