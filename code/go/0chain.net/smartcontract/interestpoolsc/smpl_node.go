package interestpoolsc

import (
	"encoding/json"

	"0chain.net/pkg/tokens"
)

//go:generate msgp -io=false -tests=false -v

type SimpleGlobalNode struct {
	MaxMint     tokens.SAS     `json:"max_mint"`
	TotalMinted tokens.SAS     `json:"total_minted"`
	MinLock     tokens.SAS     `json:"min_lock"`
	APR         float64        `json:"apr"`
	OwnerId     string         `json:"owner_id"`
	Cost        map[string]int `json:"cost"`
}

func (sgn *SimpleGlobalNode) Encode() []byte {
	buff, _ := json.Marshal(sgn)
	return buff
}

func (sgn *SimpleGlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sgn)
	return err
}
