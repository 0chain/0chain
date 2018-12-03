package state

import (
	"0chain.net/block"
	"0chain.net/transaction"
	"0chain.net/util"
)

//ContextI - a state context interface
type ContextI interface {
	GetBlock() *block.Block
	GetState() util.MerklePatriciaTrieI
	GetTransaction() *transaction.Transaction
}

//Context - a context object used to manipulate global state
type Context struct {
	block *block.Block
	state util.MerklePatriciaTrieI
	txn   *transaction.Transaction
}

//NewStateContext - create a new state context
func NewStateContext(b *block.Block, s util.MerklePatriciaTrieI, t *transaction.Transaction) *Context {
	ctx := &Context{block: b, state: s, txn: t}
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
