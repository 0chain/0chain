package smartcontract

import (
	"errors"

	sci "0chain.net/chaincore/smartcontractinterface"
	"github.com/blang/semver/v4"
)

// LatestSupportSCVersion latest supported SC version
var LatestSupportSCVersion = semver.MustParse("1.0.0")

var (
	// ErrSmartContractVersionRegistered is returned when a smart contract version already exists
	ErrSmartContractVersionRegistered   = errors.New("SmartContracts version already registered")
	ErrSmartContractVersionNotSupported = errors.New("SmartContracts version not supported")
	ErrSmartContractNotFound            = errors.New("SmartContract not found")
	ErrSmartContractRegistered          = errors.New("SmartContract already registered")
)

var smartContractsVersions = NewSmartContractsWithVersion()

// RegisterSmartContracts register the smart contracts with version
func RegisterSmartContracts(version semver.Version, scs SmartContractors) error {
	return smartContractsVersions.Register(version.String(), scs)
}

// GetSmartContractsWithVersion returns the smart contracts of given version
func GetSmartContractsWithVersion(version semver.Version) (SmartContractors, bool) {
	return smartContractsVersions.Get(version.String())
}

// GetSmartContract returns the current running smart contract by address
func GetSmartContract(version semver.Version, scAddress string) (sci.SmartContractInterface, error) {
	return smartContractsVersions.GetSmartContract(version.String(), scAddress)
}

// GetSmartContracts returns the current running smart contracts
func GetSmartContracts(version semver.Version) (SmartContractors, error) {
	scs, ok := GetSmartContractsWithVersion(version)
	if !ok {
		return nil, ErrSmartContractVersionNotSupported
	}

	return scs, nil
}

// GetSmartContractsMap returns the current running smart contracts map
func GetSmartContractsMap(version semver.Version) (map[string]sci.SmartContractInterface, error) {
	scs, ok := GetSmartContractsWithVersion(version)
	if !ok {
		return nil, ErrSmartContractVersionNotSupported
	}

	return scs.GetAll(), nil
}
