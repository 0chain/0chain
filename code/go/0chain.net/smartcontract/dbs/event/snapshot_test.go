package event

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProviderCountInSnapshot(t *testing.T) {
	t.Run("test blobber count", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		blobberCountBefore := s.BlobberCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddBlobber,
			},
		})
		require.Equal(t, blobberCountBefore+1, s.BlobberCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteBlobber,
			},
		})
		require.Equal(t, blobberCountBefore, s.BlobberCount)
	})

	t.Run("test miner count", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		minerCountBefore := s.MinerCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddMiner,
			},
		})
		require.Equal(t, minerCountBefore+1, s.MinerCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteMiner,
			},
		})
		require.Equal(t, minerCountBefore, s.MinerCount)
	})

	t.Run("test sharder count", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		sharderCountBefore := s.SharderCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddSharder,
			},
		})
		require.Equal(t, sharderCountBefore+1, s.SharderCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteSharder,
			},
		})
		require.Equal(t, sharderCountBefore, s.BlobberCount)
	})

	t.Run("test sharder count", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		sharderCountBefore := s.SharderCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddSharder,
			},
		})
		require.Equal(t, sharderCountBefore+1, s.SharderCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteSharder,
			},
		})
		require.Equal(t, sharderCountBefore, s.BlobberCount)
	})

	t.Run("test authorizer count", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		authorizerCountBefore := s.AuthorizerCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddAuthorizer,
			},
		})
		require.Equal(t, authorizerCountBefore+1, s.AuthorizerCount)

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagDeleteAuthorizer,
			},
		})
		require.Equal(t, authorizerCountBefore, s.AuthorizerCount)
	})

	t.Run("test validator count", func(t *testing.T) {
		eventDb, clean := GetTestEventDB(t)
		defer clean()

		s, err := eventDb.GetGlobal()
		require.NoError(t, err)
		validatorCountBefore := s.ValidatorCount

		s.update([]Event{
			{
				Type: TypeStats,
				Tag:  TagAddOrOverwiteValidator,
			},
		})
		require.Equal(t, validatorCountBefore+1, s.ValidatorCount)
	})
}