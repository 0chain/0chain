package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

// providerFetch extracts Provider stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func providerFetch(scID, id string, db *store.Connection, sci chain.StateContextI) (*bmp.Provider, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, providerType, id))
	if err != nil {
		if list, _ := providersFetch(AllConsumersKey, db); list != nil {
			_, _ = list.del(id, db) // sync list
		}

		return nil, err
	}

	provider := bmp.Provider{}
	if err = provider.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &provider, nil
}
