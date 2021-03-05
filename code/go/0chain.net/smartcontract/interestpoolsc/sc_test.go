package interestpoolsc

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	"0chain.net/chaincore/transaction"
)

func TestInterestPoolSmartContract_GetName(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestInterestPoolSmartContract_GetName_interest",
			want: "interest",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipsc := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			if got := ipsc.GetName(); got != tt.want {
				t.Errorf("GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterestPoolSmartContract_GetAddress(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestInterestPoolSmartContract_GetAddress",
			want: "cf8d0df9bd8cc637a4ff4e792ffe3686da6220c45f0e1103baa609f3f1751ef4",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipsc := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			if got := ipsc.GetAddress(); got != tt.want {
				t.Errorf("GetAddress() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterestPoolSmartContract_SetSC(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		sc        *smartcontractinterface.SmartContract
		bcContext smartcontractinterface.BCContextI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestInterestPoolSmartContract_SetSC_getPoolsStats",
			args: args{
				sc: &smartcontractinterface.SmartContract{
					RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
					SmartContractExecutionStats: map[string]interface{}{},
				},
				bcContext: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ipsc := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}

			ipsc.SetSC(tt.args.sc, tt.args.bcContext)

			getPoolsStats := ipsc.RestHandlers["/getPoolsStats"]
			if reflect.ValueOf(getPoolsStats).Pointer() != reflect.ValueOf(ipsc.getPoolsStats).Pointer() {
				t.Error("SetSC() personalPeriodicLimit wrong set result")
			}

			getLockConfig := ipsc.RestHandlers["/getLockConfig"]
			if reflect.ValueOf(getLockConfig).Pointer() != reflect.ValueOf(ipsc.getLockConfig).Pointer() {
				t.Error("SetSC() globalPerodicLimit wrong set result")
			}

			lockMetric := metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "lock"), nil)
			if !reflect.DeepEqual(lockMetric, ipsc.SmartContractExecutionStats["lock"]) {
				t.Error("SetSC() lock wrong set result")
			}

			unlock := metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "unlock"), nil)
			if !reflect.DeepEqual(unlock, ipsc.SmartContractExecutionStats["unlock"]) {
				t.Error("SetSC() unlock wrong set result")
			}

			updateVariables := metrics.GetOrRegisterTimer(fmt.Sprintf("sc:%v:func:%v", ipsc.ID, "updateVariables"), nil)
			if !reflect.DeepEqual(updateVariables, ipsc.SmartContractExecutionStats["updateVariables"]) {
				t.Error("SetSC() updateVariables wrong set result")
			}

		})
	}
}

func TestInterestPoolSmartContract_lock(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		t                *transaction.Transaction
		un               *UserNode
		gn               *GlobalNode
		inputData        []byte
		balances         state.StateContextI
		wantTransferPool bool
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       string
		wantErr    bool
		addr       string
		shouldBeOk bool
	}{
		{
			name: "request not formatted correctly",
			args: args{
				t:         makeTestTx1Ok(0),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(20, 100),
				inputData: newTestPoolRequestWrong(),
				balances:  newTestEmptyBalances(),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "insufficent amount to dig an interest pool",
			args: args{
				t:         makeTestTx1Ok(0),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(20, 5),
				inputData: newTestPoolRequestOK(time.Second),
				balances:  newTestEmptyBalances(),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "you have no tokens to your name",
			args: args{
				t:         makeTestTx1Ok(10),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(20, 5),
				inputData: newTestPoolRequestOK(time.Second),
				balances:  newTestEmptyBalances(),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "lock amount is greater than balance",
			args: args{
				t:         makeTestTx1Ok(10),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(20, 5),
				inputData: newTestPoolRequestOK(time.Second),
				balances:  newTestBalanceForClient1Ok(1),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "duration  is longer than max lock period ",
			args: args{
				t:         makeTestTx1Ok(10),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(20, 5),
				inputData: newTestPoolRequestOK(2 * YEAR),
				balances:  newTestBalanceForClient1Ok(20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "duration is shorter than min lock period",
			args: args{
				t:         makeTestTx1Ok(10),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(2*time.Second, 5),
				inputData: newTestPoolRequestOK(time.Second),
				balances:  newTestBalanceForClient1Ok(20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "can't mint anymore",
			args: args{
				t:         makeTestTx1Ok(10),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNode(2*time.Second, 5),
				inputData: newTestPoolRequestOK(3 * time.Second),
				balances:  newTestBalanceForClient1Ok(20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "invalid miner",
			args: args{
				t:         makeTestTx1Ok(10),
				un:        newEmptyUserNode(),
				gn:        newTestGlobalNodeWithMint(2*time.Second, 5),
				inputData: newTestPoolRequestOK(3 * time.Second),
				balances:  newTestBalanceForClient1Ok(20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "ok",
			args: args{
				t:                makeTestTx1Ok(10),
				un:               newEmptyUserNode(),
				gn:               newTestGlobalNodeWithMint(2*time.Second, 5),
				inputData:        newTestPoolRequestOK(3 * time.Second),
				balances:         newTestBalanceForClient1Ok(20),
				wantTransferPool: true,
			},
			addr:       ADDRESS,
			shouldBeOk: true,
			wantErr:    false,
			want:       "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &InterestPoolSmartContract{}
			addr := tt.addr
			if tt.addr == "" {
				addr = clientID1
			}
			sc := &smartcontractinterface.SmartContract{
				ID:                          addr,
				RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
				SmartContractExecutionStats: map[string]interface{}{},
			}
			if tt.args.wantTransferPool {
				tt.want = newTokenPoolTransferResponse(tt.args.t)
			}
			ip.SetSC(sc, nil)
			got, err := ip.lock(tt.args.t, tt.args.un, tt.args.gn, tt.args.inputData, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("lock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("lock() got = %v, want %v", got, tt.want)
			}
			if tt.shouldBeOk {

			}
		})
	}
}
