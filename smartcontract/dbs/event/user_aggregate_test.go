package event

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEventDb_updateUserAggregates(t *testing.T) {
	edb, clean := GetTestEventDB(t)
	defer clean()

	if err := edb.addPartition(0, "user_aggregates"); err != nil {
		t.Error()
	}

	aggregate := &UserAggregate{
		UserID:          "client31",
		Round:           3,
		CollectedReward: 44,
		TotalStake:      55,
		ReadPoolTotal:   66,
		WritePoolTotal:  77,
		PayedFees:       88,
		CreatedAt:       time.Time{},
	}
	aggrs := map[string]*UserAggregate{
		aggregate.UserID: aggregate,
	}
	if err := edb.addUserAggregates(aggrs); err != nil {
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
			tt.wantErr(t, edb.updateUserAggregates(tt.args.e), fmt.Sprintf("updateUserAggregates(%v)", tt.args.e))
		})
	}
}
