package state

import (
	"encoding/json"

	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

//go:generate msgp -io=false -tests=false -v

var ErrInvalidTransfer = common.NewError("invalid_transfer", "invalid transfer of state")

//Transfer - a data structure to hold state transfer from one client to another
type Transfer struct {
	ClientID   string        `json:"from"`
	ToClientID string        `json:"to"`
	Amount     currency.Coin `json:"amount"`
}

//NewTransfer - create a new transfer
func NewTransfer(fromClientID, toClientID datastore.Key, zcn currency.Coin) *Transfer {
	t := &Transfer{ClientID: fromClientID, ToClientID: toClientID, Amount: zcn}
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
