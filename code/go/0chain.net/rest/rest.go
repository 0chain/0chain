package rest

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
)

type RestHandler struct {
	SCtx state.StateContextI
}

func (rh *RestHandler) GetEventDB() *event.EventDb {
	return rh.SCtx.GetEventDB()
}

func (rh *RestHandler) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	return rh.SCtx.GetTrieNode(key, v)
}
