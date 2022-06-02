package setupsc

import (
	"fmt"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
)

type SCName int

const (
	Faucet SCName = iota
	Storage
	Multisig
	Miner
	Vesting
	Zcn
)

var (
	SCNames = []string{
		"faucet",
		"storage",
		"multisig",
		"miner",
		"vesting",
		"zcn",
	}

	SCCode = map[string]SCName{
		"faucet":   Faucet,
		"storage":  Storage,
		"multisig": Multisig,
		"miner":    Miner,
		"vesting":  Vesting,
		"zcn":      Zcn,
	}
)

//SetupSmartContracts initializes smart contract addresses
func SetupSmartContracts() {
	for _, name := range SCNames {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", name)) {
			var contract = newSmartContract(name)
			smartcontract.ContractMap[contract.GetAddress()] = contract
		}
	}
}

func newSmartContract(name string) sci.SmartContractInterface {
	code, ok := SCCode[name]
	if !ok {
		return nil
	}
	switch code {
	case Faucet:
		return faucetsc.NewFaucetSmartContract()
	case Storage:
		return storagesc.NewStorageSmartContract()
	case Multisig:
		return multisigsc.NewMultiSigSmartContract()
	case Miner:
		return minersc.NewMinerSmartContract()
	case Vesting:
		return vestingsc.NewVestingSmartContract()
	case Zcn:
		return zcnsc.NewZCNSmartContract()
	default:
		return nil
	}
}
