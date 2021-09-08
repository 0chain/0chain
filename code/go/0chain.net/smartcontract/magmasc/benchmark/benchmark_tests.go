package benchmark

import (
	bk "0chain.net/smartcontract/benchmark"
)

func Tests(bd bk.BenchData, _ bk.SignatureScheme) bk.TestSuit {
	return bk.TestSuit{
		Source: bk.Magma,
		Benchmarks: []bk.BenchTestI{
			newConsumerRegisterBenchTest(),
			newConsumerRegisterStressBenchTest(bd),
			newProviderRegisterBenchTest(),
			newProviderRegisterStressBenchTest(bd),
			newConsumerUpdateBenchTest(),
			newConsumerUpdateStressBenchTest(bd),
			newProviderUpdateBenchTest(),
			newProviderUpdateStressBenchTest(bd),
			newProviderSessionInitBenchTest(),
			newProviderSessionInitStressBenchTest(bd),
			newConsumerSessionStartBenchTest(),
			newConsumerSessionStartStressBenchTest(bd),
			newProviderDataUsageBenchTest(),
			newProviderDataUsageStressBenchTest(bd),
			newConsumerSessionStopBenchTest(),
			newConsumerSessionStopStressBenchTest(bd),
		},
	}
}
