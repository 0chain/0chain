package event

import "gorm.io/gorm"

type StakePool struct {
	gorm.Model

	ProviderId   string `json:"provider_id"`
	ProviderType int    `json:"provider_type"`
	Balance      int64
	Unstake      int64
}
