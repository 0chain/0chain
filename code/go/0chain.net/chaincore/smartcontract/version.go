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
	// scVersion is the cached smart contract version on MPT 'sc_version' node
	initOnce               sync.Once
	scVersion              scVersionWithLock
	emptySC                semver.Version
	smartContractsVersions = NewSmartContractsWithVersion()
)

// scVersionWithLock
type scVersionWithLock struct {
	v    semver.Version
	lock sync.RWMutex
}

func (scv *scVersionWithLock) Set(v semver.Version) {
	scv.lock.Lock()
	scv.v = v
	scv.lock.Unlock()
}

func (scv *scVersionWithLock) Get() semver.Version {
	scv.lock.RLock()
	v := scv.v
	scv.lock.RUnlock()
	return v
}

func (scv *scVersionWithLock) String() string {
	scv.lock.RLock()
	s := scv.v.String()
	scv.lock.RUnlock()
	return s
}

// InitSCVersionOnce initialize sc version once
func InitSCVersionOnce(version *semver.Version) {
	initOnce.Do(func() {
		SetSCVersion(*version)
	})
}

// IsSCVersionReady returns true if the scVersion is not empty
func IsSCVersionReady() bool {
	return !scVersion.Get().Equals(emptySC)
}

// GetSCVersion returns the current running smart contract version
func GetSCVersion() semver.Version {
	return scVersion.Get()
}

// SetSCVersion sets the sc version
func SetSCVersion(v semver.Version) {
	scVersion.Set(v)
}

// CanUpdateSCVersion checks if we can update the smart contract version
// return the allowed version
func CanUpdateSCVersion() (*semver.Version, bool) {
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
