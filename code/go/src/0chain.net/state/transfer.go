package state

import (
	"encoding/json"

	"0chain.net/common"
	"0chain.net/datastore"
)

var ErrInvalidTransfer = common.NewError("invalid_transfer", "invalid transfer of state")

//Transfer - a data structure to hold state transfer from one client to another
type Transfer struct {
	ClientID   datastore.Key `json:"from"`
	ToClientID datastore.Key `json:"to"`
	Amount     Balance       `json:"amount"`
}

//NewTransfer - create a new transfer
func NewTransfer(fromClientID, toClientID datastore.Key, amount Balance) *Transfer {
	t := &Transfer{ClientID: fromClientID, ToClientID: toClientID, Amount: amount}
	return t
}

func (t *Transfer) Encode() []byte {
	buff, _ := json.Marshal(t)
	return buff
}

func (t *Transfer) Decode(input []byte) error {
	err := json.Unmarshal(input, t)
	return err
}
