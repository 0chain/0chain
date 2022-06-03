package zcnsc

import (
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"0chain.net/smartcontract/rest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestZcnBenchmarkRestTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)

	require.EqualValues(
		t,
		len(GetEndpoints(rest.NewRestHandler(nil))),
		len(BenchmarkRestTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
	)
}
