package magmasc

import (
	"github.com/0chain/gorocksdb"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

// accessPointFetch extracts Provider stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func accessPointFetch(scID, id string, db *gorocksdb.TransactionDB, sci chain.StateContextI) (*zmc.AccessPoint, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, accessPointType, id))
	if err != nil {
		if list, _ := accessPointsFetch(AllAccessPointsKey, db); list != nil {
			_, _ = list.del(id, db) // sync list
		}

		return nil, err
	}

	ap := zmc.AccessPoint{}
	if err = ap.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &ap, nil
}
