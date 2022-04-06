package rest

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
)

type RestHandler struct {
	state.StateContextI
	Address string
}

func (rh *RestHandler) GetEventDb() *event.EventDb {
	return rh.GetEventDb()
}

func (rh *RestHandler) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	return rh.GetTrieNode(key, v)
}
