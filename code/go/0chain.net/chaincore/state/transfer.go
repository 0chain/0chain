package state

import (
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

var ErrInvalidTransfer = common.NewError("invalid_transfer", "invalid transfer of state")

type Transfer struct {
	Sender   datastore.Key `json:"from"`
	Receiver datastore.Key `json:"to"`
	Amount   Balance       `json:"amount"`
}

func NewTransfer(sender, receiver datastore.Key, amount Balance) *Transfer {
	t := &Transfer{Sender: sender, Receiver: receiver, Amount: amount}
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
