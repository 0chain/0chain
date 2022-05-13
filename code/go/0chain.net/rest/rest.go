package rest

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/core/logging"
	"0chain.net/rest/restinterface"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
)

type RestHandler struct {
	scAccessor restinterface.StateContextAccessor
	sCtx       state.ReadOnlyStateContextI
}

func (rh *RestHandler) GetSC() state.ReadOnlyStateContextI {
	if rh.scAccessor.GetCurrentRound() != rh.sCtx.GetBlock().Round {
		rh.sCtx = rh.scAccessor.GetROStateContext()
	}
	return rh.sCtx
}

func (rh *RestHandler) SetScAccessor(sca restinterface.StateContextAccessor) {
	rh.scAccessor = sca
}

func (rh *RestHandler) SetupRestHandlers() {
	if rh.GetSC().GetEventDB() == nil {
		logging.Logger.Warn("no event database, skipping REST handlers")
		return
	}
	storagesc.SetupRestHandler(rh)
	minersc.SetupRestHandler(rh)
	faucetsc.SetupRestHandler(rh)
	vestingsc.SetupRestHandler(rh)
	zcnsc.SetupRestHandler(rh)
}

func GetFunctionNames(address string) []string {
	switch address {
	case storagesc.ADDRESS:
		return storagesc.GetRestNames()
	case minersc.ADDRESS:
		return minersc.GetRestNames()
	case faucetsc.ADDRESS:
		return faucetsc.GetRestNames()
	case vestingsc.ADDRESS:
		return vestingsc.GetRestNames()
	case zcnsc.ADDRESS:
		return zcnsc.GetRestNames()
	default:
		return []string{}
	}
}
