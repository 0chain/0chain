package event

import (
	"testing"

	"0chain.net/smartcontract/dbs"
	"github.com/0chain/common/core/currency"
	"github.com/stretchr/testify/require"
)

func makeUserEvent(id string, balance currency.Coin) Event {
	return Event{
		Type:  TypeStats,
		Tag:   TagAddOrOverwriteUser,
		Index: id,
		Data: User{
			UserID:  id,
			Balance: balance,
		},
	}
}

func makeBlobberTotalStakeEvent(id string, totalStake currency.Coin) Event {
	return Event{
		Type:  TypeStats,
		Tag:   TagUpdateBlobberTotalStake,
		Index: id,
		Data: Blobber{
			Provider: Provider{
				ID:         id,
				TotalStake: totalStake,
			},
		},
	}
}

func makeBlobberTotalOffersEvent(id string, totalOffers currency.Coin) Event {
	return Event{
		Type:  TypeStats,
		Tag:   TagUpdateBlobberTotalOffers,
		Index: id,
		Data: Blobber{
			Provider:    Provider{ID: id},
			OffersTotal: totalOffers,
		},
	}
}

func TestMergeUserEvents(t *testing.T) {
	type expect struct {
		users  map[string]User
		others []Event
	}

	tt := []struct {
		name      string
		events    []Event
		round     int64
		blockHash string
		expect    expect
	}{
		{
			name:   "no user",
			events: []Event{},
			expect: expect{
				users: map[string]User{},
			},
		},
		{
			name:   "one user",
			events: []Event{makeUserEvent("user_1", 100)},
			expect: expect{
				users: map[string]User{
					"user_1": {UserID: "user_1", Balance: 100},
				},
			},
		},
		{
			name: "two different users",
			events: []Event{
				makeUserEvent("user_1", 100),
				makeUserEvent("user_2", 200),
			},
			expect: expect{
				users: map[string]User{
					"user_1": {UserID: "user_1", Balance: 100},
					"user_2": {UserID: "user_2", Balance: 200},
				},
			},
		},
		{
			name: "two users with same index",
			events: []Event{
				makeUserEvent("user_1", 100),
				makeUserEvent("user_1", 200),
			},
			expect: expect{
				users: map[string]User{
					"user_1": {UserID: "user_1", Balance: 200},
				},
			},
		},
		{
			name: "part of users with same index",
			events: []Event{
				makeUserEvent("user_1", 100),
				makeUserEvent("user_2", 200),
				makeUserEvent("user_3", 300),
				makeUserEvent("user_1", 200),
			},
			expect: expect{
				users: map[string]User{
					"user_1": {UserID: "user_1", Balance: 200},
					"user_2": {UserID: "user_2", Balance: 200},
					"user_3": {UserID: "user_3", Balance: 300},
				},
			},
		},
		{
			name: "part of users with same index with others",
			events: []Event{
				makeUserEvent("user_1", 100),
				makeUserEvent("user_2", 200),
				makeUserEvent("user_3", 300),
				makeUserEvent("user_1", 200),
				makeBlobberTotalStakeEvent("blobber_1", 1000),
				makeBlobberTotalStakeEvent("blobber_2", 2000),
			},
			expect: expect{
				users: map[string]User{
					"user_1": {UserID: "user_1", Balance: 200},
					"user_2": {UserID: "user_2", Balance: 200},
					"user_3": {UserID: "user_3", Balance: 300},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			um := mergeAddUsersEvents()
			others := make([]Event, 0, len(tc.events))
			for _, u := range tc.events {
				if um.filter(u) {
					continue
				}

				others = append(others, u)
			}

			ue, err := um.merge(tc.round, tc.blockHash)
			require.NoError(t, err)

			if ue == nil {
				return
			}

			users, ok := fromEvent[[]User](ue.Data)
			require.True(t, ok)

			require.Equal(t, len(tc.expect.users), len(*users))

			for _, u := range *users {
				exp, ok := tc.expect.users[u.UserID]
				require.True(t, ok)
				require.EqualValues(t, exp, u)
			}
		})
	}
}

func TestMergeBlobberTotalStakesEvents(t *testing.T) {
	type expect struct {
		blobbers map[string]Blobber
		others   []Event
	}

	tt := []struct {
		name      string
		events    []Event
		round     int64
		blockHash string
		expect    expect
	}{
		{
			name:   "no blobber",
			events: []Event{},
			expect: expect{
				blobbers: map[string]Blobber{},
			},
		},
		{
			name:   "one blobber",
			events: []Event{makeBlobberTotalStakeEvent("b_1", 100)},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1", TotalStake: 100}},
				},
			},
		},
		{
			name: "two different blobbers",
			events: []Event{
				makeBlobberTotalStakeEvent("b_1", 100),
				makeBlobberTotalStakeEvent("b_2", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1", TotalStake: 100}},
					"b_2": {Provider: Provider{ID: "b_2", TotalStake: 200}},
				},
			},
		},
		{
			name: "two blobbers with same index",
			events: []Event{
				makeBlobberTotalStakeEvent("b_1", 100),
				makeBlobberTotalStakeEvent("b_1", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1", TotalStake: 200}},
				},
			},
		},
		{
			name: "part of blobbers with same index",
			events: []Event{
				makeBlobberTotalStakeEvent("b_1", 100),
				makeBlobberTotalStakeEvent("b_2", 200),
				makeBlobberTotalStakeEvent("b_3", 300),
				makeBlobberTotalStakeEvent("b_1", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1", TotalStake: 200}},
					"b_2": {Provider: Provider{ID: "b_2", TotalStake: 200}},
					"b_3": {Provider: Provider{ID: "b_3", TotalStake: 300}},
				},
			},
		},
		{
			name: "part of blobbers with same index",
			events: []Event{
				makeBlobberTotalStakeEvent("b_1", 100),
				makeBlobberTotalStakeEvent("b_2", 200),
				makeBlobberTotalStakeEvent("b_3", 300),
				makeBlobberTotalStakeEvent("b_1", 200),
				makeUserEvent("user_1", 100),
				makeUserEvent("user_2", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1", TotalStake: 200}},
					"b_2": {Provider: Provider{ID: "b_2", TotalStake: 200}},
					"b_3": {Provider: Provider{ID: "b_3", TotalStake: 300}},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			em := mergeUpdateBlobberTotalStakesEvents()
			others := make([]Event, 0, len(tc.events))
			for _, e := range tc.events {
				if em.filter(e) {
					continue
				}

				others = append(others, e)
			}

			mergedEvent, err := em.merge(tc.round, tc.blockHash)
			require.NoError(t, err)

			if mergedEvent == nil {
				return
			}

			blobbers, ok := fromEvent[[]Blobber](mergedEvent.Data)
			require.True(t, ok)

			require.Equal(t, len(tc.expect.blobbers), len(*blobbers))

			for _, b := range *blobbers {
				exp, ok := tc.expect.blobbers[b.ID]
				require.True(t, ok)
				require.EqualValues(t, exp, b)
			}
		})
	}
}

func TestMergeBlobberTotalOffersEvents(t *testing.T) {
	type expect struct {
		blobbers map[string]Blobber
		others   []Event
	}

	tt := []struct {
		name      string
		events    []Event
		round     int64
		blockHash string
		expect    expect
	}{
		{
			name:   "no blobber",
			events: []Event{},
			expect: expect{
				blobbers: map[string]Blobber{},
			},
		},
		{
			name:   "one blobber",
			events: []Event{makeBlobberTotalOffersEvent("b_1", 100)},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1"}, OffersTotal: 100},
				},
			},
		},
		{
			name: "two different blobbers",
			events: []Event{
				makeBlobberTotalOffersEvent("b_1", 100),
				makeBlobberTotalOffersEvent("b_2", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1"}, OffersTotal: 100},
					"b_2": {Provider: Provider{ID: "b_2"}, OffersTotal: 200},
				},
			},
		},
		{
			name: "two blobbers with same index",
			events: []Event{
				makeBlobberTotalOffersEvent("b_1", 100),
				makeBlobberTotalOffersEvent("b_1", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1"}, OffersTotal: 200},
				},
			},
		},
		{
			name: "part of blobbers with same index",
			events: []Event{
				makeBlobberTotalOffersEvent("b_1", 100),
				makeBlobberTotalOffersEvent("b_2", 200),
				makeBlobberTotalOffersEvent("b_3", 300),
				makeBlobberTotalOffersEvent("b_1", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1"}, OffersTotal: 200},
					"b_2": {Provider: Provider{ID: "b_2"}, OffersTotal: 200},
					"b_3": {Provider: Provider{ID: "b_3"}, OffersTotal: 300},
				},
			},
		},
		{
			name: "part of blobbers with same index",
			events: []Event{
				makeBlobberTotalOffersEvent("b_1", 100),
				makeBlobberTotalOffersEvent("b_2", 200),
				makeBlobberTotalOffersEvent("b_3", 300),
				makeBlobberTotalOffersEvent("b_1", 200),
				makeUserEvent("user_1", 100),
				makeUserEvent("user_2", 200),
			},
			expect: expect{
				blobbers: map[string]Blobber{
					"b_1": {Provider: Provider{ID: "b_1"}, OffersTotal: 200},
					"b_2": {Provider: Provider{ID: "b_2"}, OffersTotal: 200},
					"b_3": {Provider: Provider{ID: "b_3"}, OffersTotal: 300},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			em := mergeUpdateBlobberTotalOffersEvents()
			others := make([]Event, 0, len(tc.events))
			for _, e := range tc.events {
				if em.filter(e) {
					continue
				}

				others = append(others, e)
			}

			mergedEvent, err := em.merge(tc.round, tc.blockHash)
			require.NoError(t, err)

			if mergedEvent == nil {
				return
			}

			blobbers, ok := fromEvent[[]Blobber](mergedEvent.Data)
			require.True(t, ok)

			require.Equal(t, len(tc.expect.blobbers), len(*blobbers))

			for _, b := range *blobbers {
				exp, ok := tc.expect.blobbers[b.ID]
				require.True(t, ok)
				require.EqualValues(t, exp.Provider, b.Provider)
				exp.Provider = b.Provider
				require.EqualValues(t, exp, b)
			}
		})
	}
}

func TestMergeStakePoolRewardsEvents(t *testing.T) {
	type expect struct {
		poolRewards map[string]dbs.StakePoolReward
		others      []Event
	}

	tt := []struct {
		name      string
		events    []Event
		round     int64
		blockHash string
		expect    expect
	}{
		{
			name:   "no stake pool reward",
			events: []Event{},
			expect: expect{
				poolRewards: map[string]dbs.StakePoolReward{},
			},
		},
		{
			name: "one stake pool reward",
			events: []Event{
				makeStakePoolRewardEvent(
					"b_1",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
			},
			expect: expect{
				poolRewards: map[string]dbs.StakePoolReward{
					"b_1": {
						Provider: dbs.Provider{ProviderId: "b_1"},
						Reward:   100,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 10,
							"bp_2": 20,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 10,
						},
					},
				},
			},
		},
		{
			name: "two different stake pool reward",
			events: []Event{
				makeStakePoolRewardEvent(
					"b_1",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
				makeStakePoolRewardEvent(
					"b_2",
					200,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
			},
			expect: expect{
				poolRewards: map[string]dbs.StakePoolReward{
					"b_1": {
						Provider: dbs.Provider{ProviderId: "b_1"},
						Reward:   100,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 10,
							"bp_2": 20,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 10,
						},
					},
					"b_2": {
						Provider: dbs.Provider{ProviderId: "b_2"},
						Reward:   200,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 10,
							"bp_2": 20,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 10,
						},
					},
				},
			},
		},
		{
			name: "two with ame stake pool reward index",
			events: []Event{
				makeStakePoolRewardEvent(
					"b_1",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
				makeStakePoolRewardEvent(
					"b_1",
					200,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
			},
			expect: expect{
				poolRewards: map[string]dbs.StakePoolReward{
					"b_1": {
						Provider: dbs.Provider{ProviderId: "b_1"},
						Reward:   300,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 20,
							"bp_2": 40,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 20,
						},
					},
				},
			},
		},
		{
			name: "partly with same index",
			events: []Event{
				makeStakePoolRewardEvent(
					"b_1",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
				makeStakePoolRewardEvent(
					"b_2",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
				makeStakePoolRewardEvent(
					"b_3",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
				makeStakePoolRewardEvent(
					"b_1",
					100,
					map[string]currency.Coin{
						"bp_1": 10,
						"bp_2": 20,
					},
					map[string]currency.Coin{
						"bp_1": 10,
					}),
			},
			expect: expect{
				poolRewards: map[string]dbs.StakePoolReward{
					"b_1": {
						Provider: dbs.Provider{ProviderId: "b_1"},
						Reward:   200,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 20,
							"bp_2": 40,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 20,
						},
					},
					"b_2": {
						Provider: dbs.Provider{ProviderId: "b_2"},
						Reward:   100,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 10,
							"bp_2": 20,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 10,
						},
					},
					"b_3": {
						Provider: dbs.Provider{ProviderId: "b_3"},
						Reward:   100,
						DelegateRewards: map[string]currency.Coin{
							"bp_1": 10,
							"bp_2": 20,
						},
						DelegatePenalties: map[string]currency.Coin{
							"bp_1": 10,
						},
					},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			em := mergeStakePoolRewardsEvents()
			others := make([]Event, 0, len(tc.events))
			for _, e := range tc.events {
				if em.filter(e) {
					continue
				}

				others = append(others, e)
			}

			mergedEvent, err := em.merge(tc.round, tc.blockHash)
			require.NoError(t, err)

			if mergedEvent == nil {
				return
			}

			poolRewards, ok := fromEvent[[]dbs.StakePoolReward](mergedEvent.Data)
			require.True(t, ok)

			require.Equal(t, len(tc.expect.poolRewards), len(*poolRewards))

			for _, pr := range *poolRewards {
				exp, ok := tc.expect.poolRewards[pr.ProviderId]
				require.True(t, ok)
				require.EqualValues(t, exp, pr)
			}
		})
	}
}

func makeStakePoolRewardEvent(id string, reward currency.Coin,
	delegateRewards map[string]currency.Coin, delegatePenalties map[string]currency.Coin) Event {
	return Event{
		Type:  TypeStats,
		Tag:   TagStakePoolReward,
		Index: id,
		Data: dbs.StakePoolReward{
			Provider: dbs.Provider{
				ProviderId: id,
			},
			Reward:            reward,
			DelegateRewards:   delegateRewards,
			DelegatePenalties: delegatePenalties,
		},
	}
}
