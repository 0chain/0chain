// +build integration_tests

package storagesc

import (
	"fmt"

	cstate "github.com/0chain/0chain/code/go/0chain.net/chaincore/chain/state"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/node"
	"github.com/0chain/0chain/code/go/0chain.net/chaincore/transaction"
	"github.com/0chain/0chain/code/go/0chain.net/core/util"

	crpc "github.com/0chain/0chain/code/go/0chain.net/conductor/conductrpc"
)

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *scConfig, blobber *StorageNode, all *StorageNodes,
	balances cstate.StateContextI) (err error) {

	// check for duplicates
	for _, b := range all.Nodes {
		if b.ID == blobber.ID || b.BaseURL == blobber.BaseURL {
			var existingBytes util.Serializable
			existingBytes, err = balances.GetTrieNode(blobber.GetKey(sc.ID))

			if err = blobber.validate(conf); err != nil {
				return fmt.Errorf("invalid values in request: %v", err)
			}

			return sc.updateBlobber(t, existingBytes, blobber, all)
		}
	}

	blobber.LastHealthCheck = t.CreationDate // set to now

	// the stake pool can be created by related validator
	var sp *stakePool
	sp, err = sc.getOrCreateStakePool(conf, blobber.ID,
		&blobber.StakePoolSettings, balances)
	if err != nil {
		return
	}

	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	all.Nodes.add(blobber) // add to all

	// statistic
	sc.statIncr(statAddBlobber)
	sc.statIncr(statNumberOfBlobbers)

	var (
		client = crpc.Client()
		state  = client.State()
		abe    crpc.AddBlobberEvent
	)
	abe.Sender = state.Name(crpc.NodeID(node.Self.Underlying().GetKey()))
	abe.Blobber = state.Name(crpc.NodeID(blobber.ID))
	if err = client.AddBlobber(&abe); err != nil {
		panic(err)
	}
	return
}
