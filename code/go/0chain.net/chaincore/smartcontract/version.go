package smartcontract

import (
	"errors"

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
	smartContractsVersions = NewSmartContractsWithVersion()
)

func init() {
	// TODO: move the version initialization work to the package user
	v, err := semver.Make("1.0.0")
	if err != nil {
		panic(err)
	}
	scVersion = v
}

// SetSCVersion sets the sc version
func SetSCVersion(version string) error {
	v, err := semver.Make(version)
	if err != nil {
		return err
	}

	scVersion = v
	return nil
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
