package event

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventDb_updateUserAggregates(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	if err := edb.addPartition(0, "user_aggregates"); err != nil {
		t.Error()
	}

	snap := UserSnapshot{
		UserID:          "client31",
		Round:           3,
		CollectedReward: 44,
		TotalStake:      55,
		ReadPoolTotal:   66,
		WritePoolTotal:  77,
		PayedFees:       88,
		CreatedAt:       time.Time{},
	}
	if err := edb.AddOrOverwriteUserSnapshots([]*UserSnapshot{ &snap }); err != nil {
		t.Error(err)
	}
	type args struct {
		e *blockEvents
	}
	tests := []struct {
		name    string
		args    args
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "single event",
			args: args{e: &blockEvents{events: []Event{{
				BlockNumber: 1,
				TxHash:      "qwerty",
				Tag:         TagLockReadPool,
				Index:       "qwety",
				Data: &[]ReadPoolLock{{
					Client: "client1",
					PoolId: "pool1",
					Amount: 10,
				}},
			}}}},
			wantErr: assert.NoError,
		}, {
			name: "two event",
			args: args{e: &blockEvents{events: []Event{{
				BlockNumber: 2,
				TxHash:      "qwerty21",
				Tag:         TagLockReadPool,
				Index:       "qwety21",
				Data: &[]ReadPoolLock{{
					Client: "client21",
					PoolId: "pool21",
					Amount: 10,
				}}}, {
				BlockNumber: 2,
				TxHash:      "qwerty22",
				Tag:         TagLockReadPool,
				Index:       "qwety22",
				Data: &[]ReadPoolLock{{
					Client: "client22",
					PoolId: "pool22",
					Amount: 10,
				}}},
			}}},
			wantErr: assert.NoError,
		}, {
			name: "two event with aggr",
			args: args{e: &blockEvents{events: []Event{{
				BlockNumber: 4,
				TxHash:      "qwerty31",
				Tag:         TagLockReadPool,
				Index:       "qwety21",
				Data: &[]ReadPoolLock{{
					Client: "client31",
					PoolId: "pool21",
					Amount: 10,
				}}}, {
				BlockNumber: 4,
				TxHash:      "qwerty32",
				Tag:         TagLockReadPool,
				Index:       "qwety22",
				Data: &[]ReadPoolLock{{
					Client: "client32",
					PoolId: "pool22",
					Amount: 10,
				}}},
			}}},
			wantErr: func(t assert.TestingT, err error, msgAndArgs ...interface{}) bool {
				if err := edb.Commit(); err != nil {
					return true
				}
				edb, err = edb.Begin()
				if err != nil {
					return true
				}

				a := map[string]interface{}{
					"client31": struct {
					}{},
				}
				aggregates, err := edb.GetLatestUserAggregates(a)
				return assert.Equal(t, 76, aggregates["client31"].ReadPoolTotal)
			},
		},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			edb.Store.Get().Exec("DROP TABLE IF EXISTS temp_ids")
			tt.wantErr(t, edb.updateUserAggregates(tt.args.e), fmt.Sprintf("updateUserAggregates(%v)", tt.args.e))
		})
	}
}

func TestEventDb_updateUserSnapshots(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	edb.Store.Get().Exec("DROP TABLE IF EXISTS temp_ids")

	if err := edb.addPartition(0, "user_aggregates"); err != nil {
		t.Error()
	}

	snap := UserSnapshot{
		UserID:          "test_client",
		Round:           3,
		CollectedReward: 44,
		TotalStake:      55,
		ReadPoolTotal:   66,
		WritePoolTotal:  77,
		PayedFees:       88,
		CreatedAt:       time.Time{},
	}
	if err := edb.AddOrOverwriteUserSnapshots([]*UserSnapshot{ &snap }); err != nil {
		t.Error(err)
	}

	events := &blockEvents{
		events: []Event{
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagLockReadPool,
				Index:       "qwety",
				Data: &[]ReadPoolLock{{
					Client: "test_client",
					PoolId: "test_read_pool",
					Amount: 10,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagUnlockReadPool,
				Index:       "qwety",
				Data: &[]ReadPoolLock{{
					Client: "test_client",
					PoolId: "test_read_pool",
					Amount: 5,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagLockWritePool,
				Index:       "qwety",
				Data: &[]WritePoolLock{{
					Client: "test_client",
					AllocationId: "test_allocation_id",
					Amount: 10,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagUnlockWritePool,
				Index:       "qwety",
				Data: &[]WritePoolLock{{
					Client: "test_client",
					AllocationId: "test_allocation_id",
					Amount: 5,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagLockStakePool,
				Index:       "qwety",
				Data: &[]DelegatePoolLock{{
					Client: "test_client",
					Amount: 10,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagUnlockStakePool,
				Index:       "qwety",
				Data: &[]DelegatePoolLock{{
					Client: "test_client",
					Amount: 5,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagUpdateUserPayedFees,
				Index:       "qwety",
				Data: &[]UserAggregate{{
					UserID: "test_client",
					PayedFees: 10,
				}},
			},
			{
				BlockNumber: 10,
				TxHash:      "qwerty",
				Tag:         TagUpdateUserCollectedRewards,
				Index:       "qwety",
				Data: &[]UserAggregate{{
					UserID: "test_client",
					CollectedReward: 10,
				}},
			},
			{
				BlockNumber: 11,
				TxHash:      "qwerty",
				Tag:         TagUpdateUserCollectedRewards,
				Index:       "qwety",
				Data: &[]UserAggregate{{
					UserID: "test_client_2",
					CollectedReward: 10,
				}},
			},
		},
	}

	err := edb.updateUserAggregates(events)
	require.NoError(t, err)

	snapsAfter, err := edb.GetUserSnapshotsByIds([]string{ "test_client", "test_client_2" })
	require.NoError(t, err)
	require.Equal(t, 2, len(snapsAfter))

	snap1, snap2 := snapsAfter[0], snapsAfter[1]
	if snap1.UserID == "test_client_2" {
		snap1, snap2 = snap2, snap1
	}
	assert.Equal(t, int64(10), snap1.Round)
	assert.Equal(t, snap.TotalStake + int64(5), snap1.TotalStake)
	assert.Equal(t, snap.ReadPoolTotal + int64(5), snap1.ReadPoolTotal)
	assert.Equal(t, snap.WritePoolTotal + int64(5), snap1.WritePoolTotal)
	assert.Equal(t, snap.PayedFees + int64(10), snap1.PayedFees)
	assert.Equal(t, snap.CollectedReward + int64(10), snap1.CollectedReward)

	assert.Equal(t, int64(11), snap2.Round)
	assert.Equal(t, int64(10), snap2.CollectedReward)	
}