package magmasc

import (
	"github.com/0chain/gorocksdb"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

// userFetch extracts User stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func userFetch(scID, id string, db *gorocksdb.TransactionDB, sci chain.StateContextI) (*zmc.User, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, userType, id))
	if err != nil {
		if list, _ := usersFetch(AllUsersKey, db); list != nil {
			_, _ = list.del(id, db) // sync list
		}

		return nil, err
	}

	user := zmc.User{}
	if err = user.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &user, nil
}
