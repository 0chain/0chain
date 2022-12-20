package provider

import (
	"errors"
	"fmt"

	cstate "0chain.net/chaincore/chain/state"
	"0chain.net/smartcontract/provider/factory"
	"0chain.net/smartcontract/provider/spenum"
)

var providerFactoryFactory = factory.NewDummyProviderFactoryFactory()

func SetProviderFactoryFactory(pff factory.ProviderFactoryFactory) {
	providerFactoryFactory = pff
}

func GetStakePool(id string, pType spenum.Provider, sCtx cstate.CommonStateContextI) (AbstractStakePool, error) {
	obj, err := providerFactoryFactory.ProviderFactory(sCtx).GetStakePool(id, pType)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("nil stake pool")
	}
	sp, ok := obj.(AbstractStakePool)
	if !ok {
		return nil, fmt.Errorf("not stake pool, %v", obj)
	}
	return sp, nil
}

func GetProvider(id string, pType spenum.Provider, sCtx cstate.CommonStateContextI) (AbstractProvider, error) {
	obj, err := providerFactoryFactory.ProviderFactory(sCtx).GetProvider(id, pType)
	if err != nil {
		return nil, err
	}
	if obj == nil {
		return nil, errors.New("nil provider")
	}
	provider, ok := obj.(AbstractProvider)
	if !ok {
		return nil, fmt.Errorf("not provider, %v", obj)
	}
	return provider, nil
}
