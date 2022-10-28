package storagesc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
	"0chain.net/smartcontract/stakepool/spenum"
)

var (
	blobberStakePoolPartitionsName = encryption.Hash("blobber_stakepool_partitions")
	blobberStakePoolPartitions     = newStakePoolPartition(blobberStakePoolPartitionsName)

	validatorStakePoolPartitionsName = encryption.Hash("validator_stakepool_partitions")
	validatorStakePoolPartitions     = newStakePoolPartition(validatorStakePoolPartitionsName)
)

func init() {
	// register blobber stake pool partitions
	regInitPartsFunc(func(state cstate.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, blobberStakePoolPartitionsName, 10)
		return err
	})

	regInitPartsFunc(func(state cstate.StateContextI) error {
		_, err := partitions.CreateIfNotExists(state, validatorStakePoolPartitionsName, 10)
		return err
	})
}

type stakePoolPartition struct {
	name string
}

func newStakePoolPartition(name string) *stakePoolPartition {
	return &stakePoolPartition{
		name: name,
	}
}

func (spp *stakePoolPartition) getPart(balances cstate.StateContextI) (*partitions.Partitions, error) {
	return partitions.GetPartitions(balances, spp.name)
}

func (spp *stakePoolPartition) update(balances cstate.StateContextI, key string, f func(sp *stakePool) error) error {
	part, err := spp.getPart(balances)
	if err != nil {
		return err
	}

	if err := part.Update(balances, key, func(data []byte) ([]byte, error) {
		sp := newStakePool()
		_, err := sp.UnmarshalMsg(data)
		if err != nil {
			return nil, err
		}

		if err := f(sp); err != nil {
			return nil, err
		}

		// TODO: emit sp.Save events
		return sp.MarshalMsg(nil)
	}); err != nil {
		return err
	}

	return part.Save(balances)
}

func (spp *stakePoolPartition) get(balances cstate.StateContextI, pty spenum.Provider, pid string) (*stakePool, error) {
	key := stakePoolKey(pty, pid)
	part, err := spp.getPart(balances)
	if err != nil {
		return nil, err
	}

	sp := newStakePool()
	if _, err := part.GetItem(balances, key, sp); err != nil {
		return nil, err
	}

	return sp, nil
}

func (spp *stakePoolPartition) updateArray(balances cstate.StateContextI, keys []string, f func(sps []*stakePool) error) error {
	part, err := spp.getPart(balances)
	if err != nil {
		return err
	}

	sps := make([]*stakePool, 0, len(keys))
	for _, k := range keys {
		sp := newStakePool()
		if _, err := part.GetItem(balances, k, sp); err != nil {
			return err
		}
		sps = append(sps, sp)
	}

	if err := f(sps); err != nil {
		return err
	}

	for _, sp := range sps {
		if err := part.UpdateItem(balances, sp); err != nil {
			return err
		}
	}

	return part.Save(balances)
}

func getStakePoolPartition(providerType spenum.Provider) (*stakePoolPartition, error) {
	var sPart *stakePoolPartition
	switch providerType {
	case spenum.Blobber:
		sPart = blobberStakePoolPartitions
	case spenum.Validator:
		sPart = validatorStakePoolPartitions
	default:
		return nil, fmt.Errorf("invalid provider type: %v", providerType)
	}
	return sPart, nil
}
