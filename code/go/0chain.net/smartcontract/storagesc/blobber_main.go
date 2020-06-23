// +build !integration_tests
package storagesc

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *scConfig, blobber *StorageNode, all *StorageNodes,
	balances cstate.StateContextI) (err error) {

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
	return
}
