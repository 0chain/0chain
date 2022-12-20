package factory_implementation

import (
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/provider/factory"
	"0chain.net/smartcontract/provider/spenum"
	"0chain.net/smartcontract/storagesc"
)

type providerFactoryFactory struct{}

func NewProviderFactoryFactory() factory.ProviderFactoryFactory {
	return providerFactoryFactory{}
}

func (pff providerFactoryFactory) ProviderFactory(csCtx cstate.CommonStateContextI) factory.ProviderFactory {
	return newProviderFactory(csCtx)
}

type providerFactory struct {
	sCtx cstate.CommonStateContextI
}

func newProviderFactory(sCtx cstate.CommonStateContextI) factory.ProviderFactory {
	return providerFactory{
		sCtx: sCtx,
	}
}

func (pf providerFactory) GetProvider(id string, pType spenum.Provider) (interface{}, error) {
	switch pType {
	case spenum.Blobber:
		return storagesc.GetBlobber(id, pf.sCtx)
	case spenum.Validator:
		return storagesc.GetValidator(id, pf.sCtx)
	default:
		return nil, fmt.Errorf("unsupported provider type %s", pType)
	}
}

func (pf providerFactory) GetStakePool(id string, pType spenum.Provider) (interface{}, error) {
	return storagesc.GetStakePoolAdapter(pType, id, pf.sCtx)
}
