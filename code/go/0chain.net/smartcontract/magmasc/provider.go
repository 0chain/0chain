package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

// providerFetch extracts Provider stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func providerFetch(scID, id datastore.Key, sci chain.StateContextI) (*bmp.Provider, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, id, providerType))
	if err != nil {
		return nil, err
	}

	provider := bmp.Provider{}
	if err = provider.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &provider, nil
}
