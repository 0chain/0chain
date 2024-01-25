package event

import (
	"testing"

	"github.com/go-faker/faker/v4"
	"github.com/stretchr/testify/require"
)

func buildMockBlobberSnapshot(t *testing.T, pid string) BlobberSnapshot {
	var snap BlobberSnapshot
	err := faker.FakeData(&snap)
	t.Logf("Created mock blobber snapshot: %+v", snap)
	require.NoError(t, err)

	snap.BlobberID = pid
	snap.IsKilled = false
	snap.IsShutdown = false
	return snap
}

func buildMockMinerSnapshot(t *testing.T, pid string) MinerSnapshot {
	var snap MinerSnapshot
	err := faker.FakeData(&snap)
	t.Logf("Created mock miner snapshot: %+v", snap)
	require.NoError(t, err)

	snap.MinerID = pid
	snap.IsKilled = false
	snap.IsShutdown = false
	return snap
}

func buildMockSharderSnapshot(t *testing.T, pid string) SharderSnapshot {
	var snap SharderSnapshot
	err := faker.FakeData(&snap)
	t.Logf("Created mock sharder snapshot: %+v", snap)
	require.NoError(t, err)

	snap.SharderID = pid
	snap.IsKilled = false
	snap.IsShutdown = false
	return snap
}

func buildMockValidatorSnapshot(t *testing.T, pid string) ValidatorSnapshot {
	var snap ValidatorSnapshot
	err := faker.FakeData(&snap)
	t.Logf("Created mock validator snapshot: %+v", snap)
	require.NoError(t, err)

	snap.ValidatorID = pid
	snap.IsKilled = false
	snap.IsShutdown = false
	return snap
}

func buildMockAuthorizerSnapshot(t *testing.T, pid string) AuthorizerSnapshot {
	var snap AuthorizerSnapshot
	err := faker.FakeData(&snap)
	t.Logf("Created mock authorizer snapshot: %+v", snap)
	require.NoError(t, err)

	snap.AuthorizerID = pid
	snap.IsKilled = false
	snap.IsShutdown = false
	return snap
}
