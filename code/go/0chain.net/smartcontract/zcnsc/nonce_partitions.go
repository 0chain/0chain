package zcnsc

import (
	"strconv"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/encryption"
	"0chain.net/smartcontract/partitions"
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

func partitionWZCNMintedNonce(state state.StateContextI) (*partitions.Partitions, error) {
	return partitions.CreateIfNotExists(state, wzcnMintedNoncePartitionName, wzcnMintedNoncePartitionSize)
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
