package smartcontract

import (
	"sync"

	sci "0chain.net/chaincore/smartcontractinterface"
)

//go:generate mockery -name SmartContractors --case underscore --output ./mocks
// SmartContractors is the interface that wraps the methods for accessing smart contracts
type SmartContractors interface {
	// Get returns registered smart contract by name
	Get(scAddress string) (sci.SmartContractInterface, bool)
	// Register register smart contract
	Register(scAddress string, sc sci.SmartContractInterface) error
	// GetAll returns all smart contracts map
	GetAll() map[string]sci.SmartContractInterface
}

// SmartContracts implements the SmartContractors interface
type SmartContracts struct {
	v     map[string]sci.SmartContractInterface
	mutex sync.Mutex
}

// NewSmartContracts returns a SmartContracts instance
func NewSmartContracts() *SmartContracts {
	return &SmartContracts{
		v: make(map[string]sci.SmartContractInterface),
	}
}

// Get returns registered smart contract by name
func (scs *SmartContracts) Get(scAddress string) (sc sci.SmartContractInterface, ok bool) {
	scs.mutex.Lock()
	sc, ok = scs.v[scAddress]
	scs.mutex.Unlock()
	return
}

// Register registers a smart contract
func (scs *SmartContracts) Register(scAddress string, sc sci.SmartContractInterface) error {
	scs.mutex.Lock()
	defer scs.mutex.Unlock()
	if _, ok := scs.v[scAddress]; ok {
		return ErrSmartContractRegistered
	}

	scs.v[scAddress] = sc
	return nil
}

// GetAll returns all smart contracts map
func (scs *SmartContracts) GetAll() map[string]sci.SmartContractInterface {
	scs.mutex.Lock()
	cv := make(map[string]sci.SmartContractInterface, len(scs.v))
	for k, sc := range scs.v {
		cv[k] = sc
	}
	scs.mutex.Unlock()
	return cv
}

// SmartContractsWithVersion stores all registered smart contracts with versions
type SmartContractsWithVersion struct {
	scs   map[string]SmartContractors
	mutex sync.Mutex
}

// NewSmartContractsWithVersion returns a new SmartContractsWithVersion instance
func NewSmartContractsWithVersion() *SmartContractsWithVersion {
	return &SmartContractsWithVersion{scs: make(map[string]SmartContractors)}
}

// Register the smart contracts with version
func (s *SmartContractsWithVersion) Register(version string, scs SmartContractors) error {
	s.mutex.Lock()
	_, ok := s.scs[version]
	if ok {
		return ErrSmartContractVersionRegistered
	}

	s.scs[version] = scs
	s.mutex.Unlock()

	return nil
}

// Get returns the registered smart contracts of give version
func (s *SmartContractsWithVersion) Get(version string) (scs SmartContractors, ok bool) {
	s.mutex.Lock()
	scs, ok = s.scs[version]
	s.mutex.Unlock()
	return
}

// GetSmartContract return the smart contract of given version and name
func (s *SmartContractsWithVersion) GetSmartContract(version, scAddress string) (sci.SmartContractInterface, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	scs, ok := s.scs[version]
	if !ok {
		return nil, ErrSmartContractVersionNotSupported
	}

	sc, ok := scs.Get(scAddress)
	if !ok {
		return nil, ErrSmartContractNotFound
	}

	return sc, nil
}
