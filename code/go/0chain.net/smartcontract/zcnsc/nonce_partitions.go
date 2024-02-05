package zcnsc

import (
	"strconv"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
	partitions_v1 "0chain.net/smartcontract/partitions_v_1"
	partitions_v2 "0chain.net/smartcontract/partitions_v_2"
)

const wzcnMintedNoncePartitionSize = 5

var (
	wzcnMintedNoncePartitionName = encryption.Hash(ADDRESS + ":wzcn_minted_nonce_partition")
)

//go:generate msgp -io=false -tests=false -v
type WZCNMintedNonce struct {
	Nonce int64 `msg:"nonce"`
}

// GetID implementes the partition.PartitionItem interface
func (wzcn *WZCNMintedNonce) GetID() string {
	return strconv.FormatInt(wzcn.Nonce, 10)
}

func partitionWZCNMintedNonce(balances state.StateContextI) (res partitions.Partitions, err error) {
	actError := state.WithActivation(balances, "apollo", func() error {
		res, err = partitionWZCNMintedNonce_v1(balances)
		return nil
	}, func() error {
		res, err = partitionWZCNMintedNonce_v2(balances)
		return nil
	})

	if actError != nil {
		return nil, actError
	}

	return
}
func partitionWZCNMintedNonce_v1(state state.StateContextI) (partitions.Partitions, error) {
	return partitions_v1.CreateIfNotExists(state, wzcnMintedNoncePartitionName, wzcnMintedNoncePartitionSize)
}
func partitionWZCNMintedNonce_v2(state state.StateContextI) (partitions.Partitions, error) {
	return partitions_v2.CreateIfNotExists(state, wzcnMintedNoncePartitionName, wzcnMintedNoncePartitionSize)
}

func PartitionWZCNMintedNonceAdd(state state.StateContextI, nonce int64) error {
	p, err := partitionWZCNMintedNonce(state)
	if err != nil {
		return err
	}

	err = p.Add(state, &WZCNMintedNonce{Nonce: nonce})
	if err != nil {
		return err
	}

	return p.Save(state)
}
