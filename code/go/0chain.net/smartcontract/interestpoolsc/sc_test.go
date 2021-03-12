package interestpoolsc

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/rcrowley/go-metrics"

	"0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/config"
	"0chain.net/chaincore/smartcontractinterface"
	bState "0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
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
				t:         testTxn(clientID1, 0),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 100, 10, 20),
				inputData: []byte("{test}"),
				balances:  testBalance("", 0),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "insufficent amount to dig an interest pool",
			args: args{
				t:         testTxn(clientID1, 0),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 5, 10, 20),
				inputData: testPoolRequest(time.Second),
				balances:  testBalance("", 0),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "you have no tokens to your name",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 5, 10, 20),
				inputData: testPoolRequest(time.Second),
				balances:  testBalance("", 0),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "lock amount is greater than balance",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 5, 10, 20),
				inputData: testPoolRequest(time.Second),
				balances:  testBalance(clientID1, 1),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "duration  is longer than max lock period ",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 5, 10, 20),
				inputData: testPoolRequest(2 * YEAR),
				balances:  testBalance(clientID1, 20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "duration is shorter than min lock period",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 5, 10, 2*time.Second),
				inputData: testPoolRequest(time.Second),
				balances:  testBalance(clientID1, 20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "can't mint anymore",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 5, 10, 2*time.Second),
				inputData: testPoolRequest(3 * time.Second),
				balances:  testBalance(clientID1, 20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "invalid miner",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 100, 1, 5, 0, 2*time.Second),
				inputData: testPoolRequest(3 * time.Second),
				balances:  testBalance(clientID1, 20),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "ok",
			args: args{
				t:                testTxn(clientID1, 10),
				un:               testUserNode(clientID1, nil),
				gn:               testGlobalNode(globalNode1Ok, 100, 1, 5, 0, 2*time.Second),
				inputData:        testPoolRequest(3 * time.Second),
				balances:         testBalance(clientID1, 20),
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
				tt.want = testTokenPoolTransferResponse(tt.args.t)
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
				amount := float64(bState.Balance(tt.args.t.Value))
				apr := tt.args.gn.APR
				dur := float64(3 * time.Second)
				balance := bState.Balance(tt.args.t.Value) + bState.Balance(amount*apr*dur/float64(YEAR))
				stateBalance, err := tt.args.balances.GetClientBalance(tt.args.t.ToClientID)
				if err != nil {
					t.Errorf("can not fetch balance for %v", tt.args.t.ToClientID)
				}
				if stateBalance != balance {
					t.Errorf("wrong balance for %v: now %v : should %v", tt.args.t.ToClientID, stateBalance, balance)
				}

				savedGNode, err := tt.args.balances.GetTrieNode(tt.args.gn.getKey())
				if err != nil {
					t.Errorf("can not fetch already saved global node")
				}
				if !reflect.DeepEqual(savedGNode, tt.args.gn) {
					t.Errorf("wrong saved node")
				}

				savedUNode, err := tt.args.balances.GetTrieNode(tt.args.un.getKey(tt.args.gn.ID))
				if err != nil {
					t.Errorf("can not fetch already saved user node")
				}

				if !reflect.DeepEqual(savedUNode, tt.args.un) {
					t.Errorf("wrong saved node")
				}
			}
		})
	}
}

func TestInterestPoolSmartContract_unlock(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		t         *transaction.Transaction
		un        *UserNode
		gn        *GlobalNode
		inputData []byte
		balances  state.StateContextI
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       string
		wantErr    bool
		shouldBeOk bool
	}{
		{
			name: "input not formatted correctly",
			args: args{
				t:         nil,
				un:        nil,
				gn:        nil,
				inputData: []byte("{test}"),
				balances:  nil,
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "pool doesn't exist",
			args: args{
				t:         nil,
				un:        testUserNode(clientID1, nil),
				gn:        testGlobalNode(globalNode1Ok, 100, 1, 100, 0, 10*time.Second),
				inputData: testPoolState().encode(),
				balances:  nil,
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "error emptying pool",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, testInterestPool(5, 0)),
				gn:        testGlobalNode(globalNode1Ok, 100, 1, 100, 0, 10*time.Second),
				inputData: testPoolState().encode(),
				balances:  testBalance(clientID1, 10),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "pool already empty",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, testInterestPool(0, 0)),
				gn:        testGlobalNode(globalNode1Ok, 100, 1, 100, 0, 10*time.Second),
				inputData: testPoolState().encode(),
				balances:  testBalance(clientID1, 10),
			},
			wantErr: true,
			want:    "",
		},
		{
			name: "ok",
			args: args{
				t:         testTxn(clientID1, 10),
				un:        testUserNode(clientID1, testInterestPool(0, 100)),
				gn:        testGlobalNode(globalNode1Ok, 100, 1, 100, 0, 10*time.Second),
				inputData: testPoolState().encode(),
				balances:  testBalance(clientID1, 10),
			},
			wantErr:    false,
			want:       "",
			shouldBeOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			sc := &smartcontractinterface.SmartContract{
				RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
				SmartContractExecutionStats: map[string]interface{}{},
			}
			ip.SetSC(sc, nil)
			got, err := ip.unlock(tt.args.t, tt.args.un, tt.args.gn, tt.args.inputData, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("unlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.shouldBeOk {
				tpr := &tokenpool.TokenPoolTransferResponse{
					ToClient: clientID1,
					Value:    100,
					FromPool: "new_test_pool_state"}
				tt.want = string(tpr.Encode())
			}
			if got != tt.want {
				t.Errorf("unlock() got = %v, want %v", got, tt.want)
			}

		})
	}
}

func TestInterestPoolSmartContract_updateVariables(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		t         *transaction.Transaction
		gn        *GlobalNode
		inputData []byte
		balances  state.StateContextI
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		want       string
		wantErr    bool
		shouldBeOk bool
	}{
		{
			name: "unauthorized access",
			args: args{
				t:         testTxn(clientID1, 100),
				gn:        nil,
				inputData: nil,
				balances:  nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "request not formatted correctly",
			args: args{
				t:         testTxn(owner, 100),
				gn:        nil,
				inputData: []byte("{test}"),
				balances:  nil,
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "ok",
			args: args{
				t:         testTxn(owner, 100),
				gn:        testGlobalNode(globalNode1Ok, 10, 10, 0, 10, 5),
				inputData: testGlobalNode(globalNode1Ok, 10, 20, 30, 40, 10).Encode(),
				balances:  testBalance("", 0),
			},
			want:       string(testGlobalNode(globalNode1Ok, 10, 10, 30, 40, 10).Encode()),
			wantErr:    false,
			shouldBeOk: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			if tt.shouldBeOk {
				config.SetupSmartContractConfig()
			}
			got, err := ip.updateVariables(tt.args.t, tt.args.gn, tt.args.inputData, tt.args.balances)
			if (err != nil) != tt.wantErr {
				t.Errorf("updateVariables() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("updateVariables() got = %v, want %v", got, tt.want)
			}
			if tt.shouldBeOk {
				const pfx = "smart_contracts.interestpoolsc."
				var conf = config.SmartContractConfig
				if conf.Get(pfx+"interest_rate") != tt.args.gn.APR {
					t.Errorf("wrong interest_rate")
				}
				if conf.Get(pfx+"min_lock_period") != tt.args.gn.MinLockPeriod {
					t.Errorf("wrong min_lock_period")
				}
				if conf.Get(pfx+"min_lock") != tt.args.gn.MinLock {
					t.Errorf("wrong min_lock")
				}
				if conf.Get(pfx+"max_mint") != tt.args.gn.MaxMint {
					t.Errorf("wrong max_mint")
				}
			}

		})
	}
}

func TestInterestPoolSmartContract_getUserNode(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		id       datastore.Key
		balances state.StateContextI
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *UserNode
		full   bool
	}{
		{
			name: "empty_node",
			args: args{
				balances: testBalance("", 0),
				id:       clientID1,
			},
			want: testUserNode(clientID1, nil),
		},
		{
			name: "full_node",
			args: args{
				balances: nil,
				id:       clientID1,
			},
			want: testUserNode(clientID1, nil),
			full: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.full {
				blnc := testBalance("", 0)
				blnc.InsertTrieNode(datastore.Key("new_test_pool_state"+clientID1), testUserNode(clientID1, nil))
				tt.args.balances = blnc
			}
			ip := &InterestPoolSmartContract{
				SmartContract: &smartcontractinterface.SmartContract{
					ID:                          "new_test_pool_state",
					RestHandlers:                map[string]smartcontractinterface.SmartContractRestHandler{},
					SmartContractExecutionStats: map[string]interface{}{},
				},
			}
			if got := ip.getUserNode(tt.args.id, tt.args.balances); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getUserNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInterestPoolSmartContract_getGlobalNode(t *testing.T) {
	type fields struct {
		SmartContract *smartcontractinterface.SmartContract
	}
	type args struct {
		balances state.StateContextI
		funcName string
	}

	type test struct {
		name   string
		fields fields
		args   args
		want   *GlobalNode
		before func(test) test
	}

	tests := []test{
		//{
		//	name: "empty_globalNode",
		//	args: args{
		//		balances: testBalance("",0,nil),
		//		funcName: "funcName",
		//	},
		//	want: newGlobalNodeWithConfigValues(),
		//},
		//{
		//	name: "existing_globalNode",
		//	args: args{
		//		balances: nil,
		//		funcName: "funcName",
		//	},
		//	before: func(tt test) test {
		//		b := testBalance("",0,nil)
		//		gn := newGlobalNode()
		//		b.InsertTrieNode(gn.getKey(), gn)
		//		tt.args.balances = b
		//		tt.want = gn
		//		return tt
		//	},
		//	want: nil,
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt = tt.before(tt)
			}
			ip := &InterestPoolSmartContract{
				SmartContract: tt.fields.SmartContract,
			}
			if got := ip.getGlobalNode(tt.args.balances, tt.args.funcName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getGlobalNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
