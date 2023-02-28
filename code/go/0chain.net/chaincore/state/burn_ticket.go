package state

import (
	"encoding/json"
)

type BurnTicket struct {
	EthereumAddress string `json:"ethereum_address"`
	Hash            string `json:"hash"`
	Nonce           int64  `json:"nonce"`
}

func NewBurnTicket(ethereumAddress, hash string, nonce int64) *BurnTicket {
	m := &BurnTicket{
		EthereumAddress: ethereumAddress,
		Hash:            hash,
		Nonce:           nonce,
	}
	return m
}

func (b *BurnTicket) Encode() []byte {
	buff, _ := json.Marshal(b)
	return buff
}

func (b *BurnTicket) Decode(input []byte) error {
	err := json.Unmarshal(input, b)
	return err
}
