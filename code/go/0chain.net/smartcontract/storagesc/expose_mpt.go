package storagesc

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
	"context"
	"net/url"
)

func (ssc *StorageSmartContract) GetMptKey(
	_ context.Context,
	params url.Values,
	balances state.StateContextI,
) (interface{}, error) {
	var err error
	var conf *scConfig
	if conf, err = ssc.getConfig(balances, false); err != nil {
		return nil, common.NewError("get_mpt_key",
			"can't get SC configurations: "+err.Error())
	}
	if !conf.ExposeMpt {
		return nil, common.NewError("get_mpt_key",
			"exposed mpt not enabled")
	}

	var key = params.Get("key")
	val, err := balances.GetTrieNode(key)
	if err != nil {
		return nil, common.NewErrorf("get_mpt_key",
			"get trie node %s failed: %v", key, err)
	}
	return string(val.Encode()), nil
}
