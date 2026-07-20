package event

import (
	"sort"
	"testing"

	"0chain.net/smartcontract/stakepool/spenum"
	"gorm.io/gorm/clause"

	"0chain.net/smartcontract/dbs"
	"github.com/stretchr/testify/assert"
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

func TestBuildChangedProvidersMapFromEvents(t *testing.T) {
	edb, rb := GetTestEventDB(t)
	defer rb()

	blobbers := []Blobber{
		buildMockBlobber(t, "blobber1"),
		buildMockBlobber(t, "blobber2"),
		buildMockBlobber(t, "blobber3"),
		buildMockBlobber(t, "blobber4"),
		buildMockBlobber(t, "blobber5"),
		buildMockBlobber(t, "blobber6"),
	}
	err := edb.Store.Get().Omit(clause.Associations).Create(&blobbers).Error
	require.NoError(t, err)

	miners := []Miner{
		buildMockMiner(t, OwnerId, "miner1"),
		buildMockMiner(t, OwnerId, "miner2"),
		buildMockMiner(t, OwnerId, "miner3"),
		buildMockMiner(t, OwnerId, "miner4"),
		buildMockMiner(t, OwnerId, "miner5"),
		buildMockMiner(t, OwnerId, "miner6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&miners).Error
	require.NoError(t, err)

	sharders := []Sharder{
		buildMockSharder(t, OwnerId, "sharder1"),
		buildMockSharder(t, OwnerId, "sharder2"),
		buildMockSharder(t, OwnerId, "sharder3"),
		buildMockSharder(t, OwnerId, "sharder4"),
		buildMockSharder(t, OwnerId, "sharder5"),
		buildMockSharder(t, OwnerId, "sharder6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&sharders).Error
	require.NoError(t, err)

	validators := []Validator{
		buildMockValidator(t, OwnerId, "validator1"),
		buildMockValidator(t, OwnerId, "validator2"),
		buildMockValidator(t, OwnerId, "validator3"),
		buildMockValidator(t, OwnerId, "validator4"),
		buildMockValidator(t, OwnerId, "validator5"),
		buildMockValidator(t, OwnerId, "validator6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&validators).Error
	require.NoError(t, err)

	authorizers := []Authorizer{
		buildMockAuthorizer(t, OwnerId, "authorizer1"),
		buildMockAuthorizer(t, OwnerId, "authorizer2"),
		buildMockAuthorizer(t, OwnerId, "authorizer3"),
		buildMockAuthorizer(t, OwnerId, "authorizer4"),
		buildMockAuthorizer(t, OwnerId, "authorizer5"),
		buildMockAuthorizer(t, OwnerId, "authorizer6"),
	}
	err = edb.Store.Get().Omit(clause.Associations).Create(&authorizers).Error
	require.NoError(t, err)

	events := []Event{
		{
			Tag: TagUpdateBlobberAllocatedSavedHealth,
			Data: []Blobber{
				blobbers[1],
				blobbers[2],
			},
		},
		{
			Tag: TagUpdateMiner,
			Data: []dbs.DbUpdates{{
				Id:      miners[1].ID,
				Updates: map[string]interface{}{"path": miners[1].Path},
			}},
		},
		{
			Tag: TagUpdateMiner,
			Data: []dbs.DbUpdates{{
				Id:      miners[2].ID,
				Updates: map[string]interface{}{"path": miners[2].Path},
			}},
		},
		{
			Tag: TagUpdateSharderTotalStake,
			Data: []Sharder{
				sharders[1],
				sharders[2],
			},
		},
		{
			Tag: TagAddAuthorizer,
			Data: []Authorizer{
				authorizers[1],
				authorizers[2],
			},
		},
		{
			Tag: TagAddOrOverwiteValidator,
			Data: []Validator{
				validators[1],
				validators[2],
			},
		},
		{
			Tag: TagStakePoolReward,
			Data: []dbs.StakePoolReward{
				{
					ProviderID: dbs.ProviderID{
						ID:   blobbers[0].ID,
						Type: spenum.Blobber,
					},
				},
				{
					ProviderID: dbs.ProviderID{
						ID:   miners[0].ID,
						Type: spenum.Miner,
					},
				},
			},
		},
		{
			Tag: TagStakePoolPenalty,
			Data: []dbs.StakePoolReward{
				{
					ProviderID: dbs.ProviderID{
						ID:   miners[1].ID,
						Type: spenum.Miner,
					},
				},
				{
					ProviderID: dbs.ProviderID{
						ID:   sharders[0].ID,
						Type: spenum.Sharder,
					},
				},
			},
		},
		{
			Tag: TagCollectProviderReward,
			Data: dbs.ProviderID{
				ID:   validators[4].ID,
				Type: spenum.Validator,
			},
		},
		{
			Tag: TagBlobberHealthCheck,
			Data: []dbs.DbHealthCheck{
				{
					ID: blobbers[3].ID,
				},
				{
					ID: blobbers[4].ID,
				},
			},
		},
		{
			Tag: TagMinerHealthCheck,
			Data: []dbs.DbHealthCheck{
				{
					ID: miners[3].ID,
				},
				{
					ID: miners[4].ID,
				},
			},
		},
		{
			Tag: TagSharderHealthCheck,
			Data: []dbs.DbHealthCheck{
				{
					ID: sharders[3].ID,
				},
				{
					ID: sharders[4].ID,
				},
			},
		},
		{
			Tag: TagAuthorizerHealthCheck,
			Data: []dbs.DbHealthCheck{
				{
					ID: authorizers[3].ID,
				},
				{
					ID: authorizers[4].ID,
				},
			},
		},
		{
			Tag: TagValidatorHealthCheck,
			Data: []dbs.DbHealthCheck{
				{
					ID: validators[3].ID,
				},
				{
					ID: validators[0].ID,
				},
			},
		},
		{
			Tag: TagKillProvider,
			Data: []dbs.ProviderID{
				{
					ID:   blobbers[5].ID,
					Type: spenum.Blobber,
				},
				{
					ID:   validators[5].ID,
					Type: spenum.Validator,
				},
			},
		},
		{
			Tag: TagShutdownProvider,
			Data: []dbs.ProviderID{
				{
					ID:   miners[5].ID,
					Type: spenum.Miner,
				},
				{
					ID:   sharders[5].ID,
					Type: spenum.Sharder,
				},
			},
		},
	}

	t.Run("test extractIdsFromEvents", func(t *testing.T) {

		ids, err := extractIdsFromEvents(events)
		require.NoError(t, err)

		idsLists := map[spenum.Provider][]string{
			spenum.Blobber:    make([]string, 0, len(ids[spenum.Blobber])),
			spenum.Miner:      make([]string, 0, len(ids[spenum.Miner])),
			spenum.Sharder:    make([]string, 0, len(ids[spenum.Sharder])),
			spenum.Authorizer: make([]string, 0, len(ids[spenum.Authorizer])),
			spenum.Validator:  make([]string, 0, len(ids[spenum.Validator])),
		}

		for providerType, idsList := range ids {
			for id, _ := range idsList {
				idsLists[providerType] = append(idsLists[providerType], id)
			}
		}

		assert.Equal(t, 6, len(idsLists[spenum.Blobber]))
		assert.ElementsMatch(t, []string{
			blobbers[0].ID,
			blobbers[1].ID,
			blobbers[2].ID,
			blobbers[3].ID,
			blobbers[4].ID,
			blobbers[5].ID,
		}, idsLists[spenum.Blobber])
		assert.Equal(t, 6, len(idsLists[spenum.Miner]))
		assert.ElementsMatch(t, []string{
			miners[0].ID,
			miners[1].ID,
			miners[2].ID,
			miners[3].ID,
			miners[4].ID,
			miners[5].ID,
		}, idsLists[spenum.Miner])
		assert.Equal(t, 6, len(idsLists[spenum.Sharder]))
		assert.ElementsMatch(t, []string{
			sharders[0].ID,
			sharders[1].ID,
			sharders[2].ID,
			sharders[3].ID,
			sharders[4].ID,
			sharders[5].ID,
		}, idsLists[spenum.Sharder])
		assert.Equal(t, 4, len(idsLists[spenum.Authorizer]))
		assert.ElementsMatch(t, []string{
			authorizers[1].ID,
			authorizers[2].ID,
			authorizers[3].ID,
			authorizers[4].ID,
		}, idsLists[spenum.Authorizer])
		assert.Equal(t, 6, len(idsLists[spenum.Validator]))
		assert.ElementsMatch(t, []string{
			validators[0].ID,
			validators[1].ID,
			validators[2].ID,
			validators[3].ID,
			validators[4].ID,
			validators[5].ID,
		}, idsLists[spenum.Validator])
	})

	t.Run("test GetProvidersByIds", func(t *testing.T) {
		providersFromDB, err := edb.GetProvidersByIds(spenum.Blobber, []string{
			blobbers[0].ID,
			blobbers[1].ID,
		})
		require.NoError(t, err)

		blobbersFromDB := make([]*Blobber, 0, len(providersFromDB))
		for _, provider := range providersFromDB {
			blobber, ok := provider.(*Blobber)
			require.True(t, ok)
			blobbersFromDB = append(blobbersFromDB, blobber)
		}

		assert.Equal(t, 2, len(blobbersFromDB))
		sort.Slice(blobbersFromDB, func(i, j int) bool {
			return blobbersFromDB[i].ID < blobbersFromDB[j].ID
		})
		assert.Equal(t, blobbers[0].ID, blobbersFromDB[0].ID)
		assert.Equal(t, blobbers[0].TotalServiceCharge, blobbersFromDB[0].TotalServiceCharge)
		assert.Equal(t, blobbers[0].LastHealthCheck, blobbersFromDB[0].LastHealthCheck)
		assert.Equal(t, blobbers[0].TotalStake, blobbersFromDB[0].TotalStake)
		assert.Equal(t, blobbers[1].ID, blobbersFromDB[1].ID)
		assert.Equal(t, blobbers[1].TotalServiceCharge, blobbersFromDB[1].TotalServiceCharge)
		assert.Equal(t, blobbers[1].LastHealthCheck, blobbersFromDB[1].LastHealthCheck)
		assert.Equal(t, blobbers[1].TotalStake, blobbersFromDB[1].TotalStake)
	})

	t.Run("test BuildChangedProvidersMapFromEvents", func(t *testing.T) {
		providers, err := edb.BuildChangedProvidersMapFromEvents(events)
		require.NoError(t, err)

		assert.Equal(t, 6, len(providers[spenum.Blobber]))
		assert.Equal(t, 6, len(providers[spenum.Miner]))
		assert.Equal(t, 6, len(providers[spenum.Sharder]))
		assert.Equal(t, 4, len(providers[spenum.Authorizer]))
		assert.Equal(t, 6, len(providers[spenum.Validator]))

		assert.Equal(t, blobbers[0].ID, providers[spenum.Blobber][blobbers[0].ID].GetID())
		assert.Equal(t, blobbers[1].ID, providers[spenum.Blobber][blobbers[1].ID].GetID())
		assert.Equal(t, blobbers[2].ID, providers[spenum.Blobber][blobbers[2].ID].GetID())
		assert.Equal(t, blobbers[3].ID, providers[spenum.Blobber][blobbers[3].ID].GetID())
		assert.Equal(t, blobbers[4].ID, providers[spenum.Blobber][blobbers[4].ID].GetID())
		assert.Equal(t, blobbers[5].ID, providers[spenum.Blobber][blobbers[5].ID].GetID())
		assert.Equal(t, miners[0].ID, providers[spenum.Miner][miners[0].ID].GetID())
		assert.Equal(t, miners[1].ID, providers[spenum.Miner][miners[1].ID].GetID())
		assert.Equal(t, miners[2].ID, providers[spenum.Miner][miners[2].ID].GetID())
		assert.Equal(t, miners[3].ID, providers[spenum.Miner][miners[3].ID].GetID())
		assert.Equal(t, miners[4].ID, providers[spenum.Miner][miners[4].ID].GetID())
		assert.Equal(t, miners[5].ID, providers[spenum.Miner][miners[5].ID].GetID())
		assert.Equal(t, sharders[0].ID, providers[spenum.Sharder][sharders[0].ID].GetID())
		assert.Equal(t, sharders[1].ID, providers[spenum.Sharder][sharders[1].ID].GetID())
		assert.Equal(t, sharders[2].ID, providers[spenum.Sharder][sharders[2].ID].GetID())
		assert.Equal(t, sharders[3].ID, providers[spenum.Sharder][sharders[3].ID].GetID())
		assert.Equal(t, sharders[4].ID, providers[spenum.Sharder][sharders[4].ID].GetID())
		assert.Equal(t, sharders[5].ID, providers[spenum.Sharder][sharders[5].ID].GetID())
		assert.Equal(t, authorizers[1].ID, providers[spenum.Authorizer][authorizers[1].ID].GetID())
		assert.Equal(t, authorizers[2].ID, providers[spenum.Authorizer][authorizers[2].ID].GetID())
		assert.Equal(t, authorizers[3].ID, providers[spenum.Authorizer][authorizers[3].ID].GetID())
		assert.Equal(t, authorizers[4].ID, providers[spenum.Authorizer][authorizers[4].ID].GetID())
		assert.Equal(t, validators[0].ID, providers[spenum.Validator][validators[0].ID].GetID())
		assert.Equal(t, validators[1].ID, providers[spenum.Validator][validators[1].ID].GetID())
		assert.Equal(t, validators[2].ID, providers[spenum.Validator][validators[2].ID].GetID())
		assert.Equal(t, validators[3].ID, providers[spenum.Validator][validators[3].ID].GetID())
		assert.Equal(t, validators[4].ID, providers[spenum.Validator][validators[4].ID].GetID())
		assert.Equal(t, validators[5].ID, providers[spenum.Validator][validators[5].ID].GetID())
	})
}
