package rest

import (
	"0chain.net/chaincore/chain/state"
	"0chain.net/rest/restinterface"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
)

type RestHandler struct {
	restinterface.QueryChainer
}

func NewRestHandler(c restinterface.QueryChainer) restinterface.RestHandlerI {
	return &RestHandler{QueryChainer: c}
}

func (rh *RestHandler) GetStateContext() state.QueryStateContextI {
	return rh.GetQueryStateContext()
}

func (rh *RestHandler) SetupRestHandlers() {
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
