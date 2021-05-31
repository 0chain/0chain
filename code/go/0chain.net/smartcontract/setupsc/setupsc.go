package setupsc

import (
	"fmt"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"0chain.net/smartcontract/zrc20sc"
)

type SCName int

const (
	Faucet SCName = iota
	Storage
	Zrc20
	Interest
	Multisig
	Miner
	Vesting
)

var (
	SCNames = []string{
		"faucet",
		"storage",
		"zrc20",
		"interest",
		"multisig",
		"miner",
		"vesting",
	}

	SCCode = map[string]SCName{
		"faucet":   Faucet,
		"storage":  Storage,
		"zrc20":    Zrc20,
		"interest": Interest,
		"multisig": Multisig,
		"miner":    Miner,
		"vesting":  Vesting,
	}
)

//SetupSmartContracts initialize smartcontract addresses
func SetupSmartContracts() {
	for _, name := range SCNames {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", name)) {
			var sci = newSmartContract(name)
			smartcontract.ContractMap[sci.GetAddress()] = sci
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
	case Zrc20:
		return zrc20sc.NewZRC20SmartContract()
	case Interest:
		return interestpoolsc.NewInterestPoolSmartContract()
	case Multisig:
		return multisigsc.NewMultiSigSmartContract()
	case Miner:
		return minersc.NewMinerSmartContract()
	case Vesting:
		return vestingsc.NewVestingSmartContract()
	default:
		return nil
	}
}
