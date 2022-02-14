//go:build !integration_tests
// +build !integration_tests

// todo: it's a legacy ugly approach; refactor later

package storagesc

import (
	"0chain.net/core/util"
	"0chain.net/smartcontract/partitions"
	"fmt"

	"0chain.net/smartcontract/stakepool/spenum"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/transaction"
)

// insert new blobber, filling its stake pool
func (sc *StorageSmartContract) insertBlobber(t *transaction.Transaction,
	conf *Config, blobber *StorageNode, blobbers *StorageNodes,
	balances cstate.StateContextI,
) (err error) {
	// check for duplicates
	for _, b := range blobbers.Nodes {
		if b.ID == blobber.ID || b.BaseURL == blobber.BaseURL {
			return sc.updateBlobber(t, conf, blobber, blobbers, balances)
		}
	}

	// check params
	if err = blobber.validate(conf); err != nil {
		return fmt.Errorf("invalid blobber params: %v", err)
	}

	blobber.LastHealthCheck = t.CreationDate // set to now

	_, err = balances.GetTrieNode(blobber.GetKey(sc.ID))
	if err != nil {
		if err != util.ErrValueNotPresent {
			return fmt.Errorf("failed to get blobber: %v", err)
		}

		allBlobbersList, err := getBlobbersList(balances)
		if err != nil {
			return fmt.Errorf("failed to get blobber list: %v", err)
		}
		_, err = allBlobbersList.Add(
			&partitions.BlobberNode{
				ID:  blobber.ID,
				Url: blobber.BaseURL,
			}, balances)
		if err != nil {
			return fmt.Errorf("failed to add blobber to partition: %v", err)
		}

		err = allBlobbersList.Save(balances)
		if err != nil {
			return fmt.Errorf("failed to save blobber partition: %v", err)
		}
	}

	// create stake pool
	var sp *stakePool
	sp, err = sc.getOrUpdateStakePool(conf, blobber.ID, spenum.Blobber,
		blobber.StakePoolSettings, balances)
	if err != nil {
		return fmt.Errorf("creating stake pool: %v", err)
	}

	if err = sp.save(sc.ID, t.ClientID, balances); err != nil {
		return fmt.Errorf("saving stake pool: %v", err)
	}

	// update the list
	blobbers.Nodes.add(blobber)
	if err := emitAddOrOverwriteBlobber(blobber, sp, balances); err != nil {
		return fmt.Errorf("emmiting blobber %v: %v", blobber, err)
	}

	// update statistic
	sc.statIncr(statAddBlobber)
	sc.statIncr(statNumberOfBlobbers)
	return
}
