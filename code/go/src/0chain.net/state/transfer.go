package state

import (
	"0chain.net/common"
	"0chain.net/datastore"
)

var ErrInvalidTransfer = common.NewError("invalid_transfer", "invalid transfer of state")

//Transfer - a data structure to hold state transfer from one client to another
type Transfer struct {
	ClientID   datastore.Key
	ToClientID datastore.Key
	Amount     Balance
}

//NewTransfer - create a new transfer
func NewTransfer(fromClientID, toClientID datastore.Key, amount Balance) *Transfer {
	t := &Transfer{ClientID: fromClientID, ToClientID: toClientID, Amount: amount}
	return t
}
