package rest

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
)

type RestHandler struct {
	SCtx state.StateContextI
}

func (rh *RestHandler) SetStateContext(sCtx state.StateContextI) {
	rh.SCtx = sCtx
}

func (rh *RestHandler) GetEventDB() *event.EventDb {
	return rh.SCtx.GetEventDB()
}

func (rh *RestHandler) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	return rh.SCtx.GetTrieNode(key, v)
}

func (rh *RestHandler) GetBlock() *block.Block {
	return rh.SCtx.GetBlock()
}

func (rh *RestHandler) SetupRestHandlers() {
	if rh.GetEventDB() == nil {
		logging.Logger.Warn("no event database, skipping REST handlers")
		return
	}
	//SetupStorageRestHandler(rh)
	//SetupMinerRestHandler(rh)
	faucetsc.SetupRestHandler(rh)
	interestpoolsc.SetupRestHandler(rh)
	vestingsc.SetupRestHandler(rh)
	zcnsc.SetupRestHandler(rh)
}
