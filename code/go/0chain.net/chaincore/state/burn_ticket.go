package state

import (
	"encoding/json"
)

type BurnTicket struct {
	UserID          string `json:"user_id"`
	EthereumAddress string `json:"ethereum_address"`
	Hash            string `json:"hash"`
	Nonce           int64  `json:"nonce"`
}

func NewBurnTicket(userID, ethereumAddress, hash string, nonce int64) *BurnTicket {
	m := &BurnTicket{
		UserID:          userID,
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
