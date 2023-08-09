package storagesc

import (
	"testing"

	"0chain.net/core/common"
	"0chain.net/smartcontract/benchmark"
	"0chain.net/smartcontract/benchmark/mocks"
	"0chain.net/smartcontract/rest"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

const (
	numberDevelopmentEndpoints = 12
	numberDuplicatedTests      = 3
	numberMissingTests         = 0
)

// TestStorageBenchmarkRestTests
// Checks that we have benchmarks for all endpoints.
// If this test fails either add a new benchmark or increment missing tests.
func TestStorageBenchmarkRestTests(t *testing.T) {
	mockSigScheme := &mocks.SignatureScheme{}
	mockSigScheme.On("SetPublicKey", mock.Anything).Return(nil)
	mockSigScheme.On("SetPrivateKey", mock.Anything).Return()
	mockSigScheme.On("Sign", mock.Anything).Return("", nil)
	common.ConfigRateLimits()

	numberEndpoints := len(GetEndpoints(rest.NewRestHandler(nil))) +
		numberDuplicatedTests - numberDevelopmentEndpoints - numberMissingTests
	require.Equal(
		t,
		numberEndpoints,
		len(BenchmarkRestTests(benchmark.MockBenchData, mockSigScheme).Benchmarks),
	)
}
