package state

import (
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

var ErrInvalidMint = common.NewError("invalid_mint", "invalid minter")

type Mint struct {
	Minter   datastore.Key `json:"minter"`
	Receiver datastore.Key `json:"to"`
	Amount   Balance       `json:"amount"`
}

func NewMint(minter, receiver datastore.Key, amount Balance) *Mint {
	m := &Mint{Minter: minter, Receiver: receiver, Amount: amount}
	return m
}

func (m *Mint) Encode() []byte {
	buff, _ := json.Marshal(m)
	return buff
}

func (m *Mint) Decode(input []byte) error {
	err := json.Unmarshal(input, m)
	return err
}
