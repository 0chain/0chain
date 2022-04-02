package storagesc

import (
	"context"
	"net/url"

	"0chain.net/chaincore/chain/state"
)

func (ssc *StorageSmartContract) GetMptKey(
	_ context.Context,
	params url.Values,
	balances state.StateContextI,
) (interface{}, error) {
	return "not supported", nil
}
