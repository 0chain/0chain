package zcnsc

import (
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestZcnBenchmarkTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)

	zsc := &ZCNSmartContract{
		smartcontractinterface.NewSC(ADDRESS),
		make(map[string]smartContractFunction),
	}
	zsc.InitSC()

	removeTests := 1

	require.EqualValues(
		t,
		len(zsc.smartContractFunctions)-removeTests,
		len(BenchmarkTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
	)
}
