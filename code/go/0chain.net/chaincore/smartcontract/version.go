package smartcontract

import (
	"errors"
	"sync"

	sci "0chain.net/chaincore/smartcontractinterface"
	"github.com/blang/semver/v4"
)

var (
	// ErrSmartContractVersionRegistered is returned when a smart contract version already exists
	ErrSmartContractVersionRegistered   = errors.New("SmartContracts version already registered")
	ErrSmartContractVersionNotSupported = errors.New("SmartContracts version not supported")
	ErrSmartContractNotFound            = errors.New("SmartContract not found")
	ErrSmartContractRegistered          = errors.New("SmartContract already registered")
)

var (
	// scVersion is the cached smart contract version on MPT '/sc_version' node
	scVersion              semver.Version
	vLock                  sync.RWMutex
	smartContractsVersions = NewSmartContractsWithVersion()
)

func setSCVersion(v semver.Version) {
	vLock.Lock()
	scVersion = v
	vLock.Unlock()
}

func init() {
	// TODO: move the version initialization work to the package user
	v, err := semver.Make("1.0.0")
	if err != nil {
		panic(err)
	}
	setSCVersion(v)
}

// SetSCVersion sets the sc version
func SetSCVersion(version string) error {
	v, err := semver.Make(version)
	if err != nil {
		return err
	}

	setSCVersion(v)
	return nil
}

// GetSCVersion returns the current running smart contract version
func GetSCVersion() semver.Version {
	vLock.RLock()
	v := scVersion
	vLock.RUnlock()
	return v
}

// CanSCVersionUpdate checks if we can update the smart contract version
// return the allowed version
func CanSCVersionUpdate() (*semver.Version, bool) {
	// TODO: implement this
	v2, err := semver.New("2.0.0")
	if err != nil {
		panic(err)
	}
	return v2, true
}

// RegisterSmartContracts register the smart contracts with version
func RegisterSmartContracts(version string, scs SmartContractors) error {
	return smartContractsVersions.Register(version, scs)
}

// GetSmartContractsWithVersion returns the smart contracts of given version
func GetSmartContractsWithVersion(version string) (SmartContractors, bool) {
	return smartContractsVersions.Get(version)
}

// GetSmartContract returns the current running smart contract by address
func GetSmartContract(scAddress string) (sci.SmartContractInterface, error) {
	return smartContractsVersions.GetSmartContract(scVersion.String(), scAddress)
}

// GetSmartContracts returns the current running smart contracts
func GetSmartContracts() (SmartContractors, error) {
	scs, ok := GetSmartContractsWithVersion(scVersion.String())
	if !ok {
		return nil, ErrSmartContractVersionNotSupported
	}

	return scs, nil
}

// GetSmartContractsMap returns the current running smart contracts map
func GetSmartContractsMap() (map[string]sci.SmartContractInterface, error) {
	scs, ok := GetSmartContractsWithVersion(scVersion.String())
	if !ok {
		return nil, ErrSmartContractVersionNotSupported
	}

	return scs.GetAll(), nil
}
