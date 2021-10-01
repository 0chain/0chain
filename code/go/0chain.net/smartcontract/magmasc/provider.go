package magmasc

import (
	"github.com/0chain/gorocksdb"
	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

// providerFetch extracts Provider stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func providerFetch(scID, id string, db *gorocksdb.TransactionDB, sci chain.StateContextI) (*zmc.Provider, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, providerType, id))
	if err != nil {
		if list, _ := providersFetch(AllProvidersKey, db); list != nil {
			_, _ = list.del(id, db) // sync list
		}

		return nil, err
	}

	provider := zmc.Provider{}
	if err = provider.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &provider, nil
}
