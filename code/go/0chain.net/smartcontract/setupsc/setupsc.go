package setupsc

import (
	"fmt"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/logging"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zcnsc"
	"go.uber.org/zap"
)

type SCName int

const (
	Faucet SCName = iota
	Storage
	Interest
	Multisig
	Miner
	Vesting
	Zcn
)

var (
	SCNames = []string{
		"faucet",
		"storage",
		"interest",
		"multisig",
		"miner",
		"vesting",
		"zcn",
	}

	SCCode = map[string]SCName{
		"faucet":   Faucet,
		"storage":  Storage,
		"interest": Interest,
		"multisig": Multisig,
		"miner":    Miner,
		"vesting":  Vesting,
		"zcn":      Zcn,
	}
)

//SetupSmartContracts initializes smart contract addresses
func SetupSmartContracts() {
	scs := smartcontract.NewSmartContracts()

	for _, name := range SCNames {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", name)) {
			var contract = newSmartContract(name)

			if err := scs.Register(contract.GetAddress(), contract); err != nil {
				logging.Logger.Panic("register smart contract failed",
					zap.Error(err),
					zap.String("name", name))
				return
			}
		}
	}

	// register the current smart contract as version 1.0.0
	if err := smartcontract.RegisterSmartContracts("1.0.0", scs); err != nil {
		logging.Logger.Panic("register smart contracts failed",
			zap.String("version", "1.0.0"),
			zap.Error(err))
		return
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
	case Interest:
		return interestpoolsc.NewInterestPoolSmartContract()
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
