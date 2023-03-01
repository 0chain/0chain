package storagesc

import (
	"testing"

	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const extraStats = 7

func TestStorageBenchmarkTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)

	ssc := NewStorageSmartContract()

	require.EqualValues(
		t,
		len(ssc.GetExecutionStats())-extraStats,
		len(BenchmarkTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
	)
}
