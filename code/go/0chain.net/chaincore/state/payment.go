package state

import (
	"0chain.net/core/util"
	"encoding/json"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

type Valuable interface {
	util.Serializable
	Value() Balance
}

//todo: refactor Transfer/Mint structs into something uniform

var ErrInvalidTransfer = common.NewError("invalid_transfer", "invalid transfer of state")
var ErrInvalidMint = common.NewError("invalid_mint", "invalid minter")

type Transfer struct {
	Sender   datastore.Key `json:"from"`
	Receiver datastore.Key `json:"to"`
	Amount   Balance       `json:"amount"`
}

type Mint struct {
	Minter   datastore.Key `json:"minter"`
	Receiver datastore.Key `json:"to"`
	Amount   Balance       `json:"amount"`
}

func NewTransfer(sender, receiver datastore.Key, amount Balance) *Transfer {
	t := &Transfer{Sender: sender, Receiver: receiver, Amount: amount}
	return t
}

func NewMint(minter, receiver datastore.Key, amount Balance) *Mint {
	m := &Mint{Minter: minter, Receiver: receiver, Amount: amount}
	return m
}

func (t *Transfer) Encode() []byte {
	buff, _ := json.Marshal(t)
	return buff
}

func (m *Mint) Encode() []byte {
	buff, _ := json.Marshal(m)
	return buff
}

func (m *Mint) Decode(input []byte) error {
	err := json.Unmarshal(input, m)
	return err
}

func (t *Transfer) Decode(input []byte) error {
	err := json.Unmarshal(input, t)
	return err
}

func (m* Mint) Value() Balance {
	return m.Amount
}

func (t* Transfer) Value() Balance {
	return t.Amount
}