// +build integration_tests

package storagesc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/util"

	crpc "0chain.net/conductor/conductrpc"
)

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *scConfig, blobber *StorageNode, blobbers *StorageNodes,
	balances cstate.StateContextI
) (err error) {
	// check for duplicates
	for _, b := range blobbers.Nodes {
		if b.ID == blobber.ID || b.BaseURL == blobber.BaseURL {
			return sc.updateBlobber(t, conf, blobber, blobbers, balances)
		}
	}

	// check blobber values
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
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

	blobbers.Nodes.add(blobber) // add to all

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
