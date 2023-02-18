package entity

import (
	"encoding/binary"
	"encoding/json"
	"reflect"
)

// BurnTicketDetails represents client's burn ticket details
type BurnTicketDetails struct {
	Hash  string `json:"hash"`
	Nonce int64  `json:"nonce"`
}

func (b *BurnTicketDetails) MarshalMsg(dst []byte) ([]byte, error) {
	var err error
	dst, err = json.Marshal(b)
	return dst, err
}

func (b *BurnTicketDetails) UnmarshalMsg(src []byte) ([]byte, error) {
	return src, json.Unmarshal(src, b)
}

func (b *BurnTicketDetails) Msgsize() int {
	return binary.Size(reflect.ValueOf(*b))
}
