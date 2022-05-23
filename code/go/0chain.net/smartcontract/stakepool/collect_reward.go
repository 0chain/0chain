package stakepool

import (
	"encoding/json"

	"0chain.net/smartcontract/stakepool/spenum"
)

// CollectRewardRequest uniquely defines a stake pool.
type CollectRewardRequest struct {
	ProviderId   string          `json:"provider_id"`
	ProviderType spenum.Provider `json:"provider_type"`
	PoolId       string          `json:"pool_id"`
}

func (spr *CollectRewardRequest) Decode(p []byte) error {
	return json.Unmarshal(p, spr)
}

func (spr *CollectRewardRequest) Encode() []byte {
	bytes, _ := json.Marshal(spr)
	return bytes
}
