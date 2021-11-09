package storagesc

import (
	"context"
	"net/url"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/common"
)

func (ssc *StorageSmartContract) GetMptKey(
	_ context.Context,
	params url.Values,
	balances state.RestStateContextI,
) (interface{}, error) {
	var err error
	var conf *scConfig
	if conf, err = ssc.getConfigReadOnly(balances); err != nil {
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
