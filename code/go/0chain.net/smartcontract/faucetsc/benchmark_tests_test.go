package faucetsc

import (
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBenchmarkTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)

	fsc := NewFaucetSmartContract()

	require.EqualValues(
		t,
		len(BenchmarkTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
		len(fsc.GetExecutionStats()),
	)
}
