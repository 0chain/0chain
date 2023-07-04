package event

import (
	"testing"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm/clause"
)

func TestBlobberAggregates(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()


	blobbers := []Blobber{
		{
			Provider: Provider{
				ID: "blobber1",
				TotalStake: 100,
			},
			BaseURL: "http://blobber1.com",
		},
		{
			Provider: Provider{
				ID: "blobber2",
				TotalStake: 200,
			},
			BaseURL: "http://blobber2.com",
		},
		{
			Provider: Provider{
				ID: "blobber3",
				TotalStake: 300,
			},
			BaseURL: "http://blobber3.com",
		},
		{
			Provider: Provider{
				ID: "blobber4",
				TotalStake: 400,
			},
			BaseURL: "http://blobber4.com",
		},
		{
			Provider: Provider{
				ID: "blobber5",
				TotalStake: 500,
			},
			BaseURL: "http://blobber5.com",
		},
		{
			Provider: Provider{
				ID: "blobber6",
				TotalStake: 600,
			},
			BaseURL: "http://blobber6.com",
		},
		{
			Provider: Provider{
				ID: "blobber7",
				TotalStake: 700,
			},
			BaseURL: "http://blobber7.com",
		},
		{
			Provider: Provider{
				ID: "blobber8",
				TotalStake: 800,
			},
			BaseURL: "http://blobber8.com",
		},
		{
			Provider: Provider{
				ID: "blobber9",
				TotalStake: 9000,
			},
			BaseURL: "http://blobber9.com",
		},
		{
			Provider: Provider{
				ID: "blobber10",
				TotalStake: 10000,
			},
			BaseURL: "http://blobber10.com",
		},
	}
	err := edb.Get().Create(&blobbers).Error
	require.NoError(t, err)
	
	round := 10
	events := []Event{
		{
			Tag:  TagAddBlobber,
			Data: []Blobber{
				{Provider: Provider{ID: "blobber1"}},
				{Provider: Provider{ID: "blobber2"}},
			},
		},
		{
			Tag: TagStakePoolReward,
			Data: []dbs.StakePoolReward{
				{
					ProviderID: dbs.ProviderID{ID: "blobber3"},
				},
				{
					ProviderID: dbs.ProviderID{ID: "blobber2"},
				},
			},
		},
		{
			Tag: TagBlobberHealthCheck,
			Data: []dbs.DbHealthCheck{
				{
					ID: "blobber1",
				},
				{
					ID: "blobber5",
				},
			},
		},
		{
			Tag: TagShutdownProvider,
			Data: []dbs.ProviderID{
				{ ID: "blobber4", Type: spenum.Blobber },
				{ ID: "blobber6", Type: spenum.Blobber },
				{ ID: "minerx", Type: spenum.Miner },
			},
		},
		{
			Tag: TagCollectProviderReward,
			Index: "blobber7",
		},
	}

	err = updateProviderAggregates[Blobber](edb, &blockEvents{events: events, round: int64(round)})
	require.NoError(t, err)

	var blobberAggregates []BlobberAggregate
	err = edb.Get().Find(&blobberAggregates).Error
	require.NoError(t, err)

	require.Equal(t, 7, len(blobberAggregates))
	requiredBlobbers := []string{"blobber1", "blobber2", "blobber3", "blobber4", "blobber5", "blobber6", "blobber7"}
	for i, blobber := range requiredBlobbers {
		var aggregate BlobberAggregate
		err = edb.Get().Where("blobber_id = ?", blobber).First(&aggregate).Error
		require.NoError(t, err)
		require.Equal(t, blobber, aggregate.BlobberID)
		require.Equal(t, int64(round), aggregate.Round)
		require.Equal(t, ((i+1) * 100), int(aggregate.TotalStake))
	}

	// Update 1, 4 and 7
	newBlobbers := []Blobber{
		{
			Provider: Provider{
				ID: "blobber1",
				TotalStake: 1000,
			},
		},
		{
			Provider: Provider{
				ID: "blobber4",
				TotalStake: 4000,
			},
		},
		{
			Provider: Provider{
				ID: "blobber7",
				TotalStake: 7000,
			},
		},
	}
	err = edb.Get().Model(&Blobber{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake"}),
	}).Create(newBlobbers).Error
	require.NoError(t, err)

	round = 20
	events = []Event{
		{
			Tag:  TagAddBlobber,
			Data: []Blobber{
				{Provider: Provider{ID: "blobber1"}},
				{Provider: Provider{ID: "blobber4"}},
			},
		},
		{
			Tag: TagStakePoolReward,
			Data: []dbs.StakePoolReward{
				{
					ProviderID: dbs.ProviderID{ID: "blobber4"},
				},
				{
					ProviderID: dbs.ProviderID{ID: "blobber7"},
				},
			},
		},
		{
			Tag: TagAddBlobber,
			Data: []Blobber{
				{Provider: Provider{ID: "blobber7"}},
				{Provider: Provider{ID: "blobber10"}},
			},
		},
		{
			Tag: TagMintReward,
			Data: RewardMint{
				ProviderID: "blobber9",
				ProviderType: "blobber",
			},
		},
	}

	err = updateProviderAggregates[Blobber](edb, &blockEvents{events: events, round: int64(round)})
	require.NoError(t, err)

	err = edb.Get().Find(&blobberAggregates).Error
	require.NoError(t, err)

	require.Equal(t, 12, len(blobberAggregates))

	err = edb.Get().Where("round = ?", round).Find(&blobberAggregates).Error
	require.NoError(t, err)

	require.Equal(t, 5, len(blobberAggregates))
	requiredBlobbersStake := map[string]int64{"blobber1": 1000, "blobber4": 4000, "blobber7": 7000, "blobber9": 9000, "blobber10": 10000}
	for blobber, stake := range requiredBlobbersStake {
		var aggregate BlobberAggregate
		err = edb.Get().Where("blobber_id = ? and round = ?", blobber, round).First(&aggregate).Error
		require.NoError(t, err)
		require.Equal(t, blobber, aggregate.BlobberID)
		require.Equal(t, int64(round), aggregate.Round)
		require.Equal(t, stake, int64(aggregate.TotalStake))
	}
}

// replicate the test above for miners
func TestUpdateMinerAggregates(t *testing.T) {
	edb, cleanup := GetTestEventDB(t)
	defer cleanup()

	// Add 3 miners
	miners := []Miner{
		{Provider: Provider{ID: "miner1", TotalStake: 100}},
		{Provider: Provider{ID: "miner2", TotalStake: 200}},
		{Provider: Provider{ID: "miner3", TotalStake: 300}},
		{Provider: Provider{ID: "miner4", TotalStake: 400}},
		{Provider: Provider{ID: "miner5", TotalStake: 500}},
		{Provider: Provider{ID: "miner6", TotalStake: 600}},
		{Provider: Provider{ID: "miner7", TotalStake: 700}},
		{Provider: Provider{ID: "miner8", TotalStake: 800}},
		{Provider: Provider{ID: "miner9", TotalStake: 9000}},
		{Provider: Provider{ID: "miner10", TotalStake: 10000}},
	}
	err := edb.Get().Create(&miners).Error
	require.NoError(t, err)

	// Add 2 miners
	round := 10
	events := []Event{
		{
			Tag:  TagAddMiner,
			Data: []Miner{
				{Provider: Provider{ID: "miner1"}},
				{Provider: Provider{ID: "miner2"}},
			},
		},
		{
			Tag: TagStakePoolReward,
			Data: []dbs.StakePoolReward{
				{
					ProviderID: dbs.ProviderID{ID: "miner3"},
				},
				{
					ProviderID: dbs.ProviderID{ID: "miner2"},
				},
			},
		},
		{
			Tag: TagShutdownProvider,
			Data: []dbs.ProviderID{
				{ID: "miner4", Type: spenum.Miner},
				{ID: "miner5", Type: spenum.Miner},
				{ID: "miner6", Type: spenum.Miner},
				{ID: "blobber1", Type: spenum.Blobber},
			},
		},
		{
			Tag: TagCollectProviderReward,
			Index: "miner7",
		},
	}

	err = updateProviderAggregates[Miner](edb, &blockEvents{events: events, round: int64(round)})
	require.NoError(t, err)

	var minerAggregates []MinerAggregate
	err = edb.Get().Find(&minerAggregates).Error
	require.NoError(t, err)

	require.Equal(t, 7, len(minerAggregates))
	requiredMiners := []string{"miner1", "miner2", "miner3"}
	for i, miner := range requiredMiners {
		var aggregate MinerAggregate
		err = edb.Get().Where("miner_id = ?", miner).First(&aggregate).Error
		require.NoError(t, err)
		require.Equal(t, miner, aggregate.MinerID)
		require.Equal(t, int64(round), aggregate.Round)
		require.Equal(t, ((i+1) * 100), int(aggregate.TotalStake))
	}

	// Update 1, 4 and 7
	newMiners := []Miner{
		{
			Provider: Provider{
				ID: "miner1",
				TotalStake: 1000,
			},
		},
		{
			Provider: Provider{
				ID: "miner4",
				TotalStake: 4000,
			},
		},
		{
			Provider: Provider{
				ID: "miner7",
				TotalStake: 7000,
			},
		},
	}
	err = edb.Get().Model(&Miner{}).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"total_stake"}),
	}).Create(newMiners).Error
	require.NoError(t, err)

	round = 20
	events = []Event{
		{
			Tag:  TagAddMiner,
			Data: []Miner{
				{Provider: Provider{ID: "miner1"}},
				{Provider: Provider{ID: "miner4"}},
			},
		},
		{
			Tag: TagStakePoolReward,
			Data: []dbs.StakePoolReward{
				{
					ProviderID: dbs.ProviderID{ID: "miner4"},
				},
				{
					ProviderID: dbs.ProviderID{ID: "miner7"},
				},
			},
		},
		{
			Tag: TagAddMiner,
			Data: []Miner{
				{Provider: Provider{ID: "miner7"}},
				{Provider: Provider{ID: "miner10"}},
			},
		},
		{
			Tag: TagMintReward,
			Data: RewardMint{
				ProviderID: "miner9",
				ProviderType: "miner",
			},
		},
	}

	err = updateProviderAggregates[Miner](edb, &blockEvents{events: events, round: int64(round)})
	require.NoError(t, err)

	err = edb.Get().Find(&minerAggregates).Error
	require.NoError(t, err)

	require.Equal(t, 12, len(minerAggregates))

	err = edb.Get().Where("round = ?", round).Find(&minerAggregates).Error
	require.NoError(t, err)

	require.Equal(t, 5, len(minerAggregates))
	requiredMinersStake := map[string]int64{"miner1": 1000, "miner4": 4000, "miner7": 7000, "miner9": 9000, "miner10": 10000}
	for miner, stake := range requiredMinersStake {
		var aggregate MinerAggregate
		err = edb.Get().Where("miner_id = ? and round = ?", miner, round).First(&aggregate).Error
		require.NoError(t, err)
		require.Equal(t, miner, aggregate.MinerID)
		require.Equal(t, int64(round), aggregate.Round)
		require.Equal(t, stake, int64(aggregate.TotalStake))
	}
}

