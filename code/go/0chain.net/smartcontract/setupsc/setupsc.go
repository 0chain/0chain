package setupsc

import (
	"log"

	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/viper"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/magmasc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
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
	Magma
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
		magmasc.Name,
	}

	SCCode = map[string]SCName{
		"faucet":     Faucet,
		"storage":    Storage,
		"zrc20":      Zrc20,
		"interest":   Interest,
		"multisig":   Multisig,
		"miner":      Miner,
		"vesting":    Vesting,
		magmasc.Name: Magma,
	}
)

// SetupSmartContracts initialize smart contract addresses.
func SetupSmartContracts() {
	for _, name := range SCNames {
		if viper.GetBool("development.smart_contract." + name) {
			sc := newSmartContract(name)
			if sc == nil {
				log.Panic("setup smart contracts failed")
			}
			smartcontract.ContractMap[sc.GetAddress()] = sc
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
	case Magma:
		config.SetupSmartContractConfig()
		msc := magmasc.NewMagmaSmartContract()
		cfg := config.SmartContractConfig.Sub("smart_contracts." + magmasc.Name)
		if err := msc.Setup(cfg); err != nil {
			log.Println(err)
			return nil
		}
		return msc

	default:
		return nil
	}
}
