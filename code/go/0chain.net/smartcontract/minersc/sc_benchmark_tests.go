package minersc

import (
	sc "0chain.net/smartcontract/benchmark"
	"github.com/spf13/viper"
)

func BenchmarkTests(
	vi *viper.Viper,
	clients []string,
	keys []string,
	blobbers []string,
	allocations []string,
) []sc.BenchTest {
	return []sc.BenchTest{}
}
