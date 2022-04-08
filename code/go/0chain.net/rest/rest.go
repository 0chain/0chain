package rest

import (
	"errors"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"0chain.net/core/util"
	"0chain.net/smartcontract/dbs/event"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
)

type RestHandler struct {
	SCtx state.ReadOnlyStateContextI
}

func (rh *RestHandler) SetStateContext(sCtx state.ReadOnlyStateContextI) {
	rh.SCtx = sCtx
}

func (rh *RestHandler) GetEventDB() *event.EventDb {
	if rh.SCtx == nil {
		return nil
	}
	return rh.SCtx.GetEventDB()
}

func (rh *RestHandler) GetTrieNode(key datastore.Key, v util.MPTSerializable) error {
	if rh.SCtx == nil {
		return errors.New("state context object nil")
	}
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
	//storagesc.SetupRestHandler(rh)
	minersc.SetupRestHandler(rh)
	faucetsc.SetupRestHandler(rh)
	interestpoolsc.SetupRestHandler(rh)
	vestingsc.SetupRestHandler(rh)
	zcnsc.SetupRestHandler(rh)
}

func (rh *RestHandler) GetFunctionNames(address string) []string {
	switch address {
	case storagesc.ADDRESS:
		return storagesc.GetRestNames()
	case minersc.ADDRESS:
		return minersc.GetRestNames()
	case faucetsc.ADDRESS:
		return faucetsc.GetRestNames()
	case interestpoolsc.ADDRESS:
		return interestpoolsc.GetRestNames()
	case vestingsc.ADDRESS:
		return vestingsc.GetRestNames()
	case zcnsc.ADDRESS:
		return zcnsc.GetRestNames()
	default:
		return []string{}
	}
}
