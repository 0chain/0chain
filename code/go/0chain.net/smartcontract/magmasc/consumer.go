package magmasc

import (
	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

// consumerFetch extracts Consumer stored in state.StateContextI
// or returns error if blockchain state does not contain it.
func consumerFetch(scID, id string, db *store.Connection, sci chain.StateContextI) (*bmp.Consumer, error) {
	data, err := sci.GetTrieNode(nodeUID(scID, consumerType, id))
	if err != nil {
		if list, _ := consumersFetch(AllConsumersKey, db); list != nil {
			_, _ = list.del(id, db) // sync list
		}

		return nil, err
	}

	consumer := bmp.Consumer{}
	if err = consumer.Decode(data.Encode()); err != nil {
		return nil, errDecodeData.Wrap(err)
	}

	return &consumer, nil
}
