package event

import (
	"testing"

	"0chain.net/smartcontract/stakepool/spenum"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/require"
)

func TestTagKillProvider(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	round := int64(7)

	minerIds := createMiners(t, edb, 2)
	createSharders(t, edb, 2)
	err := edb.addBlobbers([]Blobber{
		{
			Provider: Provider{ID: "blobber one"},
			BaseURL:  "one.com",
		}, {
			Provider: Provider{ID: "blobber two"},
			BaseURL:  "two.com",
		},
	})

	killEvents := []Event{
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagKillProvider,
			Index:       "blobber two",
			Data:        dbs.ProviderID{ID: "blobber two", Type: spenum.Blobber},
		},
		{
			BlockNumber: round,
			TxHash:      "2",
			Type:        TypeStats,
			Tag:         TagKillProvider,
			Index:       "blobber two",
			Data:        dbs.ProviderID{ID: "blobber two", Type: spenum.Blobber},
		},
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagKillProvider,
			Index:       minerIds[0],
			Data:        dbs.ProviderID{ID: minerIds[0], Type: spenum.Miner},
		},
		{
			BlockNumber: round,
			Type:        TypeStats,
			Tag:         TagKillProvider,
			Index:       minerIds[1],
			Data:        dbs.ProviderID{ID: minerIds[1], Type: spenum.Miner},
		},
	}
	events, err := mergeEvents(round, "", killEvents)
	require.NoError(t, err)
	require.Len(t, events, 1)
	require.Len(t, events[0].Data, 3)

	require.NoError(t, edb.addStat(events[0]))

	var miners []Miner
	edb.Get().Find(&miners)
	for _, miner := range miners {
		require.True(t, miner.IsKilled)
	}

	var blobbers []Blobber
	edb.Get().Find(&blobbers)
	for _, blobber := range blobbers {
		if blobber.ID == "blobber two" {
			require.True(t, blobber.IsKilled)
		} else {
			require.False(t, blobber.IsKilled)
		}
	}
}

func TestProvidersSetBoolean(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	minerIds := createMiners(t, edb, 2)
	createSharders(t, edb, 2)
	err := edb.addBlobbers([]Blobber{
		{
			Provider: Provider{ID: "blobber one"},
			BaseURL:  "one.com",
		}, {
			Provider: Provider{ID: "blobber two"},
			BaseURL:  "two.com",
		},
	})
	require.NoError(t, err)
	providers := []dbs.ProviderID{
		{ID: minerIds[0], Type: spenum.Miner},
		{ID: minerIds[1], Type: spenum.Miner},
		{ID: "1", Type: spenum.Sharder},
		{ID: "blobber two", Type: spenum.Blobber},
	}
	err = edb.providersSetBoolean(providers, "is_shutdown", true)
	require.NoError(t, err)

	var miners []Miner
	edb.Get().Find(&miners)
	for _, miner := range miners {
		require.True(t, miner.IsShutdown)
	}

	var blobbers []Blobber
	edb.Get().Find(&blobbers)
	for _, blobber := range blobbers {
		if blobber.ID == "blobber two" {
			require.True(t, blobber.IsShutdown)
		} else {
			require.False(t, blobber.IsShutdown)
		}
	}
}

func TestUpdateProvidersHealthCheck(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	err := edb.addBlobbers([]Blobber{
		{
			Provider: Provider{ID: "one"},
			BaseURL:  "one.com",
		}, {
			Provider: Provider{ID: "two"},
			BaseURL:  "two.com",
		},
	})

	updates := []dbs.DbHealthCheck{
		{
			ID:              "one",
			LastHealthCheck: 37,
			Downtime:        11,
		},
	}

	err = edb.updateProvidersHealthCheck(updates, "blobbers")
	require.NoError(t, err)

	var blobbers []Blobber
	edb.Get().Find(&blobbers)
	require.Equal(t, len(blobbers), 2)

}
