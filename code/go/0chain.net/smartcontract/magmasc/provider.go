package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

// providerFetch extracts Provider stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func providerFetch(scID, key string, sci chain.StateContextI) (*bmp.Provider, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, providerType, key))
	if err != nil {
		return nil, err
	}

	provider := bmp.Provider{}
	if err = provider.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &provider, nil
}
