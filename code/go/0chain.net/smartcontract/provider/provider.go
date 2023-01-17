package provider

import (
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/stakepool/spenum"
)

func GetKey(id string) datastore.Key {
	return "provider:" + id
}

type Provider struct {
	ID           string          `json:"id" validate:"hexadecimal,len=64"`
	ProviderType spenum.Provider `json:"provider_type"`
}
