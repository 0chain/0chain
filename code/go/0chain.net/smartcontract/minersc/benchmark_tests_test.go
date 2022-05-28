package minersc

import (
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

const ScStatsNotFunctionCalls = 7

func TestBenchmarkTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)

	msc := NewMinerSmartContract()

	require.EqualValues(
		t,
		len(msc.GetExecutionStats())-ScStatsNotFunctionCalls,
		len(BenchmarkTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
	)
}
