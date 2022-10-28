package storagesc

import (
	"errors"

	state "0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
)

var ErrUnknownProvider = errors.New("unknown provider type")

var (
	blobbersPartitionName = encryption.Hash("blobber_partitions")
	blobbersPartition     = newProviderPartition(blobbersPartitionName)
)

const (
	blobbersPartitionSize = 10
)

func init() {
	// register blobber stake pool partitions
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, blobbersPartitionName, 10)
		return err
	})
}

type providerPartitions struct {
	name string
}

func newProviderPartition(name string) *providerPartitions {
	return &providerPartitions{
		name: name,
	}
}

func (pp *providerPartitions) get(balances state.StateContextI, id string, v partitions.PartitionItem) error {
	part, err := partitions.GetPartitions(balances, pp.name)
	if err != nil {
		return err
	}

	_, err = part.GetItem(balances, providerKey(id), v)
	return err
}

func (pp *providerPartitions) update(balances state.StateContextI, key string, f func(data []byte) ([]byte, error)) error {
	part, err := partitions.GetPartitions(balances, pp.name)
	if err != nil {
		return err
	}

	if err := part.Update(balances, key, f); err != nil {
		return err
	}

	return part.Save(balances)
}

func (pp *providerPartitions) getPart(balances state.StateContextI) (*partitions.Partitions, error) {
	return partitions.GetPartitions(balances, pp.name)
}

func providerKey(id string) string {
	return ADDRESS + id
}
