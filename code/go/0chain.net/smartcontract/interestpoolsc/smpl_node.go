package interestpoolsc

import (
	"encoding/json"

	"0chain.net/chaincore/state"
)

//go:generate msgp -io=false -tests=false -v

type SimpleGlobalNode struct {
	MaxMint     state.Balance `json:"max_mint"`
	TotalMinted state.Balance `json:"total_minted"`
	MinLock     state.Balance `json:"min_lock"`
	APR         float64       `json:"apr"`
	OwnerId     string        `json:"owner_id"`
}

func (sgn *SimpleGlobalNode) Encode() []byte {
	buff, _ := json.Marshal(sgn)
	return buff
}

func (sgn *SimpleGlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sgn)
	return err
}
