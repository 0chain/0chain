package state

import (
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

//go:generate msgp -io=false -tests=false -v

var ErrInvalidTransfer = common.NewError("invalid_transfer", "invalid transfer of state")

//Transfer - a data structure to hold state transfer from one client to another
type Transfer struct {
	ClientID   string `json:"from"`
	ToClientID string `json:"to"`
	Amount     int64  `json:"amount"`
}

//NewTransfer - create a new transfer
func NewTransfer(fromClientID, toClientID datastore.Key, amount int64) *Transfer {
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
