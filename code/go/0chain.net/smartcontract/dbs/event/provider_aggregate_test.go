package event

import (
	"testing"

	"0chain.net/smartcontract/dbs"
	"0chain.net/smartcontract/stakepool/spenum"
	"github.com/stretchr/testify/require"
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
				TotalStake: 900,
			},
			BaseURL: "http://blobber9.com",
		},
		{
			Provider: Provider{
				ID: "blobber10",
				TotalStake: 1000,
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
			Tag: TagMintReward,
			Data: RewardMint{
				ProviderID: "blobber4",
				ProviderType: "blobber",
			},
		},
		{
			Tag: TagMintReward,
			Data: RewardMint{
				ProviderID: "minerx",
				ProviderType: "miner",
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
}