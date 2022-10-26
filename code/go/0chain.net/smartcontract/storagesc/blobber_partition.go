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

func init() {
	// register blobber stake pool partitions
	regInitPartsFunc(func(state state.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, blobbersPartitionName, 20)
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

	return part.GetItem(balances, providerKey(id), v)
}

func (pp *providerPartitions) update(balances state.StateContextI, id string, f func(data []byte) ([]byte, error)) error {
	part, err := partitions.GetPartitions(balances, pp.name)
	if err != nil {
		return err
	}

	if err := part.Update(balances, providerKey(id), f); err != nil {
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
