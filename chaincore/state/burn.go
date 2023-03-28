package state

import (
	"encoding/json"

	"github.com/0chain/common/core/currency"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

var ErrInvalidBurn = common.NewError("invalid_burn", "invalid burner")

type Burn struct {
	Burner datastore.Key `json:"burner"`
	Amount currency.Coin `json:"amount"`
}

func NewBurn(burner datastore.Key, amount currency.Coin) *Burn {
	m := &Burn{Burner: burner, Amount: amount}
	return m
}

func (b *Burn) Encode() []byte {
	buff, _ := json.Marshal(b)
	return buff
}

func (b *Burn) Decode(input []byte) error {
	err := json.Unmarshal(input, b)
	return err
}
