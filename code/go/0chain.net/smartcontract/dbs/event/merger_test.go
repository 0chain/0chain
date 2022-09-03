package event

import (
	"testing"

	"0chain.net/chaincore/currency"
	"github.com/stretchr/testify/require"
)

func makeUserEvent(id string, balance currency.Coin) Event {
	return Event{
		Type:  int(TypeStats),
		Tag:   int(TagAddOrOverwriteUser),
		Index: id,
		Data: User{
			UserID:  id,
			Balance: balance,
		},
	}
}

func makeBlobberTotalStakeEvent(id string, totalStake currency.Coin) Event {
	return Event{
		Type:  int(TypeStats),
		Tag:   int(TagUpdateBlobberTotalStake),
		Index: id,
		Data: Blobber{
			BlobberID:  id,
			TotalStake: totalStake,
		},
	}
}

func makeBlobberTotalOffersEvent(id string, totalOffers currency.Coin) Event {
	return Event{
		Type:  int(TypeStats),
		Tag:   int(TagUpdateBlobberTotalOffers),
		Index: id,
		Data: Blobber{
			BlobberID:   id,
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
			um := newUserEventsMerger()
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
					"b_1": {BlobberID: "b_1", TotalStake: 100},
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
					"b_1": {BlobberID: "b_1", TotalStake: 100},
					"b_2": {BlobberID: "b_2", TotalStake: 200},
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
					"b_1": {BlobberID: "b_1", TotalStake: 300},
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
					"b_1": {BlobberID: "b_1", TotalStake: 300},
					"b_2": {BlobberID: "b_2", TotalStake: 200},
					"b_3": {BlobberID: "b_3", TotalStake: 300},
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
					"b_1": {BlobberID: "b_1", TotalStake: 300},
					"b_2": {BlobberID: "b_2", TotalStake: 200},
					"b_3": {BlobberID: "b_3", TotalStake: 300},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			em := newBlobberTotalStakesEventsMerger()
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
				exp, ok := tc.expect.blobbers[b.BlobberID]
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
					"b_1": {BlobberID: "b_1", OffersTotal: 100},
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
					"b_1": {BlobberID: "b_1", OffersTotal: 100},
					"b_2": {BlobberID: "b_2", OffersTotal: 200},
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
					"b_1": {BlobberID: "b_1", OffersTotal: 300},
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
					"b_1": {BlobberID: "b_1", OffersTotal: 300},
					"b_2": {BlobberID: "b_2", OffersTotal: 200},
					"b_3": {BlobberID: "b_3", OffersTotal: 300},
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
					"b_1": {BlobberID: "b_1", OffersTotal: 300},
					"b_2": {BlobberID: "b_2", OffersTotal: 200},
					"b_3": {BlobberID: "b_3", OffersTotal: 300},
				},
			},
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			em := newBlobberTotalOffersEventsMerger()
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
				exp, ok := tc.expect.blobbers[b.BlobberID]
				require.True(t, ok)
				require.EqualValues(t, exp, b)
			}
		})
	}
}
