package rest

import (
	"context"
	"net/url"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
)

type RestHandler struct {
	state.StateContextI
	params url.Values
	ctx    context.Context
}

func (rh *RestHandler) GetEventDb() *event.EventDb {
	return rh.GetEventDb()
}

func (rh *RestHandler) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	return rh.GetTrieNode(key, v)
}
