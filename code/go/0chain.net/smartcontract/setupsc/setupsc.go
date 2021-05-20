package setupsc

import (
	"fmt"

	"github.com/spf13/viper"

	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/faucetsc"
	"0chain.net/smartcontract/interestpoolsc"
	"0chain.net/smartcontract/minersc"
	"0chain.net/smartcontract/multisigsc"
	"0chain.net/smartcontract/storagesc"
	"0chain.net/smartcontract/vestingsc"
	"0chain.net/smartcontract/zrc20sc"
)

type SCName string

const (
	Faucet   SCName = "faucet"
	Storage  SCName = "storage"
	Zrc20    SCName = "zrc20"
	Interest SCName = "interest"
	Multisig SCName = "multisig"
	Miner    SCName = "miner"
	Vesting  SCName = "vesting"
)

var scs = []sci.SmartContractInterface{
	&faucetsc.FaucetSmartContract{}, &storagesc.StorageSmartContract{},
	&zrc20sc.ZRC20SmartContract{}, &interestpoolsc.InterestPoolSmartContract{},
	&multisigsc.MultiSigSmartContract{},
	&minersc.MinerSmartContract{},
	&vestingsc.VestingSmartContract{},
}

//SetupSmartContracts initialize smartcontract addresses
func SetupSmartContracts() {
	for _, sc := range scs {
		if viper.GetBool(fmt.Sprintf("development.smart_contract.%v", sc.GetName())) {
			sc.InitSC()
			smartcontract.ContractMap[sc.GetAddress()] = sc
		}
	}
}

type smartContractFactorys struct {
}

func NewSmartContractFactory() sci.SmartContractFactoryI {
	return &smartContractFactorys{}
}

func (scf smartContractFactorys) NewSmartContract(name string) (sci.SmartContractInterface, *sci.SmartContract) {
	switch SCName(name) {
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
		return nil, nil
	}
}
