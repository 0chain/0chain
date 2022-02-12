package stakepool

import (
	"encoding/json"
)

type CollectRewardRequest struct {
	ProviderType Provider `json:"provider_type"`
	PoolId       string   `json:"pool_id"`
}

func (spr *CollectRewardRequest) Decode(p []byte) error {
	return json.Unmarshal(p, spr)
}
