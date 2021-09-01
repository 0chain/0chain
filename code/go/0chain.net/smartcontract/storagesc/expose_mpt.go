package storagesc

import (
	"context"
	"net/url"

	"0chain.net/chaincore/chain/state"
	"github.com/0chain/errors"
)

func (ssc *StorageSmartContract) GetMptKey(
	_ context.Context,
	params url.Values,
	balances state.StateContextI,
) (interface{}, error) {
	var err error
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, false); err != nil {
		return nil, errors.Wrap(err, errors.New("get_mpt_key", "can't get SC configurations"))

	}
	if !conf.ExposeMpt {
		return nil, errors.New("get_mpt_key",
			"exposed mpt not enabled")
	}

	var key = params.Get("key")
	val, err := balances.GetTrieNode(key)
	if err != nil {
		return nil, errors.Newf("get_mpt_key",
			"get trie node %s failed: %v", key, err)
	}
	return string(val.Encode()), nil
}
