package minersc

import (
	sci "0chain.net/chaincore/smartcontractinterface"
	sc "0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/faucetsc"
	"github.com/spf13/viper"
)

func BenchmarkTests(
	vi *viper.Viper,
	clients []string,
	keys []string,
	blobbers []string,
	allocations []string,
) []sc.BenchTest {
	var _ = NewFaucetSmartContract()MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
	}
	return []sc.BenchTest{}
}
