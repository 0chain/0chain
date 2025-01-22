package minersc

import (
	"0chain.net/chaincore/smartcontract"
	sci "0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestMinerBenchmarkTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)

	removedTests := 3

	var msc = &MinerSmartContract{
		SmartContract: sci.NewSC(ADDRESS),
		bcContext:     &smartcontract.BCContext{},
	}
	msc.smartContractFunctions = make(map[string]smartContractFunction)
	msc.initSC()

	require.EqualValues(
		t,
		len(msc.smartContractFunctions)-removedTests,
		len(BenchmarkTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
	)
}
