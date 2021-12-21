package interestpoolsc

import (
	"0chain.net/core/datastore"
	"encoding/json"

	"0chain.net/chaincore/state"
)

type SimpleGlobalNode struct {
	MaxMint     state.Balance `json:"max_mint"`
	TotalMinted state.Balance `json:"total_minted"`
	MinLock     state.Balance `json:"min_lock"`
	APR         float64       `json:"apr"`
	OwnerId     datastore.Key `json:"owner_id"`
}

func (sgn *SimpleGlobalNode) Encode() []byte {
	buff, _ := json.Marshal(sgn)
	return buff
}

func (sgn *SimpleGlobalNode) Decode(input []byte) error {
	err := json.Unmarshal(input, sgn)
	return err
}
