//go:build !integration_tests
// +build !integration_tests

// todo: it's a legacy ugly approach; refactor later

package storagesc

import (
	"encoding/json"
	"fmt"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/dbs/event"
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

	data, _ := json.Marshal(dbs.DbUpdates{
		Id: t.ClientID,
		Updates: map[string]interface{}{
			"total_stake": int64(sp.stake()),
		},
	})
	balances.EmitEvent(event.TypeStats, event.TagUpdateBlobber, t.ClientID, string(data))

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
