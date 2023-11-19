package stores

import (
	"fmt"
	"sync"

	"0chain.net/conductor/types"
)

type (
	ProviderType = types.ProviderType
	Aggregate    = types.Aggregate
	Monotonicity = types.Monotonicity

	AggregateStore struct {
		data map[ProviderType]map[string][]Aggregate
		lock sync.RWMutex
	}
)

const (
	Miner      = types.Miner
	Sharder    = types.Sharder
	Blobber    = types.Blobber
	Validator  = types.Validator
	Authorizer = types.Authorizer
	User       = types.User
	Global     = types.Global
)

var store AggregateStore

func init() {
	store = AggregateStore{}
	store.lock = sync.RWMutex{}
	store.data = make(map[ProviderType]map[string][]Aggregate)
	store.data[Sharder] = make(map[string][]Aggregate)
	store.data[Miner] = make(map[string][]Aggregate)
	store.data[Blobber] = make(map[string][]Aggregate)
	store.data[Validator] = make(map[string][]Aggregate)
	store.data[Authorizer] = make(map[string][]Aggregate)
	store.data[User] = make(map[string][]Aggregate)
	store.data[Global] = make(map[string][]Aggregate)
}

func GetAggregateStore() *AggregateStore {
	return &store
}

func (s *AggregateStore) Add(agg Aggregate, ptype ProviderType, pid string) error {
	_, err := getProviderIdStore(ptype, pid, false) // Used to check if it exists
	if err != nil {
		return err
	}

	store.data[ptype][pid] = append(store.data[ptype][pid], agg)
	return nil
}

func (s *AggregateStore) GetLatest(ptype ProviderType, pid string, key string) (Aggregate, error) {
	aggProviderIdStore, err := getProviderIdStore(ptype, pid, true)
	if err != nil {
		return nil, err
	}

	return aggProviderIdStore[len(aggProviderIdStore)-1], nil
}

func getProviderIdStore(ptype ProviderType, pid string, mustGet bool) ([]Aggregate, error) {
	aggProviderTypeStore, ok := store.data[ptype]
	if !ok {
		return nil, fmt.Errorf("unknown aggregate provider type")
	}

	aggProviderIdStore, ok := aggProviderTypeStore[pid]
	if !ok {
		if mustGet {
			return nil, types.ErrNoStoredAggregates
		}
		aggProviderIdStore = make([]Aggregate, 0)
		store.data[ptype][pid] = aggProviderIdStore
	}

	return aggProviderIdStore, nil
}
