package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

// consumerFetch extracts Consumer stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func consumerFetch(scID, id datastore.Key, sci chain.StateContextI) (*bmp.Consumer, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, id, consumerType))
	if err != nil {
		return nil, err
	}

	consumer := bmp.Consumer{}
	if err = consumer.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &consumer, nil
}
