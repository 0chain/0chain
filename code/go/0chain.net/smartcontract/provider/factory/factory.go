package factory

import (
	"errors"

	cstate "0chain.net/chaincore/chain/state"

	"0chain.net/smartcontract/provider/spenum"
)

type ProviderFactoryFactory interface {
	ProviderFactory(cstate.CommonStateContextI) ProviderFactory
}

type ProviderFactory interface {
	GetProvider(string, spenum.Provider) (interface{}, error)
	GetStakePool(string, spenum.Provider) (interface{}, error)
}

type dummyProviderFactoryFactory struct{}

func (_ dummyProviderFactoryFactory) ProviderFactory(cstate.CommonStateContextI) ProviderFactory {
	return NewDummyProviderFactory()
}

func NewDummyProviderFactoryFactory() ProviderFactoryFactory {
	return dummyProviderFactoryFactory{}
}

type dummyProviderFactory struct{}

func NewDummyProviderFactory() ProviderFactory {
	return dummyProviderFactory{}
}

func (_ dummyProviderFactory) GetProvider(_ string, _ spenum.Provider) (interface{}, error) {
	return nil, errors.New("dummy factory")
}

func (_ dummyProviderFactory) GetStakePool(_ string, _ spenum.Provider) (interface{}, error) {
	return nil, errors.New("dummy factory")
}
