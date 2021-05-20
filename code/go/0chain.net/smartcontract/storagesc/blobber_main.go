// +build !integration_tests
// todo: it's a legacy ugly approach; refactor later

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

	// check config
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid values in request: %v", err)
	}

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
	return
}
