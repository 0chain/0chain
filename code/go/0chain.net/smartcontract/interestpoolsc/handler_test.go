package interestpoolsc

import (
	"context"
	"net/url"
	"reflect"
	"testing"
	"time"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/core/common"
)

func TestInterestPoolSmartContract_getPoolsStats(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		ctx      context.Context
		params   url.Values
		balances state.StateContextI
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "case: no pools exist",
			fields: fields{SmartContract: &smartcontractinterface.SmartContract{
				ID:                          "",
				RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
				SmartContractExecutionStats: map[string]interface{}{},
			}},
			args: args{
				ctx: context.Background(),
				params: url.Values{
					"client_id": []string{clientID1},
				},
				balances: testBalance(clientID1, 100),
			},
			wantErr: true,
			want:    nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			got, err := ip.getPoolsStats(tt.args.ctx, tt.args.params, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPoolsStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPoolsStats() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterestPoolSmartContract_getPoolStats(t *testing.T) {
	now := time.Now()
	sec10 := 10 * time.Second
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		pool *interestPool
		t    time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *poolStat
		wantErr bool
	}{
		{
			name: "case: ok",
			fields: fields{SmartContract: &smartcontractinterface.SmartContract{
				ID:                          "",
				RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
				SmartContractExecutionStats: map[string]interface{}{},
			}},
			args: args{
				pool: testInterestPool(10, 100),
				t:    now.Add(sec10),
			},
			wantErr: false,
			want: &poolStat{
				ID:           "new_test_pool_state",
				StartTime:    timeNow,
				Duartion:     sec10,
				TimeLeft:     (sec10 - now.Add(sec10).Sub(common.ToTime(timeNow))),
				Locked:       now.Add(sec10).Sub(common.ToTime(timeNow)) < sec10,
				APR:          0,
				TokensEarned: 0,
				Balance:      100,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			got, err := ip.getPoolStats(tt.args.pool, tt.args.t)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPoolStats() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getPoolStats() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterestPoolSmartContract_getLockConfig(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		ctx      context.Context
		params   url.Values
		balances state.StateContextI
	}
	notEmptyBlnc := func() *testBalances {
		b := testBalance("", 0)
		gn := newGlobalNode()
		b.InsertTrieNode(gn.getKey(), gn)
		return b
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name: "ok",
			fields: fields{SmartContract: &smartcontractinterface.SmartContract{
				ID:                          "",
				RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
				SmartContractExecutionStats: map[string]interface{}{},
			}},
			args: args{
				ctx:      context.Background(),
				params:   url.Values{},
				balances: notEmptyBlnc(),
			},
			wantErr: false,
			want: testGlobalNode(
				ADDRESS,
				0, 0, 0,
				0, 0, "",
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			got, err := ip.getLockConfig(tt.args.ctx, tt.args.params, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("getLockConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getLockConfig() got = %v, want %v", got, tt.want)
			}
		})
	}
}
