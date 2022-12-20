package provider

import (
	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/smartcontract/provider/spenum"
)

//go:generate msgp -v -io=false -tests=false

func StakePoolKey(p spenum.Provider, id string) datastore.Key {
	return p.String() + ":stakepool:" + id
}

type AbstractStakePool interface {
	GetPools() map[string]*DelegatePool
	HasStakePool(user string) bool
	LockPool(txn *transaction.Transaction, providerType spenum.Provider, providerId datastore.Key, status spenum.PoolStatus, balances cstate.StateContextI) (string, error)
	EmitStakeEvent(providerType spenum.Provider, providerID string, balances cstate.StateContextI) error
	Save(providerType spenum.Provider, providerID string,
		balances cstate.StateContextI) error
	GetSettings() Settings
	Empty(sscID, poolID, clientID string, balances cstate.StateContextI) error
	UnlockPool(clientID string, providerType spenum.Provider, providerId datastore.Key, balances cstate.StateContextI) (string, error)
}
