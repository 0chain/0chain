package magmasc

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/rcrowley/go-metrics"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/smartcontractinterface"
	tx "0chain.net/chaincore/transaction"
)

func TestMain(m *testing.M) {
	code := m.Run()

	// clean up
	path := filepath.Join("/tmp", rootPath)
	if err := os.RemoveAll(path); err != nil {
		log.Println("cannot clean up path: " + path + " - please remove it manually")
	}

	os.Exit(code)
}

func Test_NewMagmaSmartContract(t *testing.T) {
	t.Parallel()

	msc := &MagmaSmartContract{SmartContract: smartcontractinterface.NewSC(zmc.Address)}

	// Magma smart contract REST handlers
	msc.RestHandlers[zmc.SessionRP] = msc.sessionAccepted
	msc.RestHandlers[zmc.VerifySessionAcceptedRP] = msc.sessionAcceptedVerify
	msc.RestHandlers[zmc.IsSessionExistRP] = msc.sessionExist
	msc.RestHandlers[zmc.GetAllConsumersRP] = msc.allConsumers
	msc.RestHandlers[zmc.GetAllProvidersRP] = msc.allProviders
	msc.RestHandlers[zmc.ConsumerRegisteredRP] = msc.consumerExist
	msc.RestHandlers[zmc.ConsumerFetchRP] = msc.consumerFetch
	msc.RestHandlers[zmc.ProviderMinStakeFetchRP] = msc.providerMinStakeFetch
	msc.RestHandlers[zmc.ProviderRegisteredRP] = msc.providerExist
	msc.RestHandlers[zmc.ProviderFetchRP] = msc.providerFetch
	msc.RestHandlers[zmc.AccessPointFetchRP] = msc.accessPointFetch
	msc.RestHandlers[zmc.AccessPointRegisteredRP] = msc.accessPointExist
	msc.RestHandlers[zmc.AccessPointMinStakeFetchRP] = msc.accessPointMinStakeFetch
	msc.RestHandlers[zmc.RewardPoolExistRP] = msc.rewardPoolExist
	msc.RestHandlers[zmc.RewardPoolFetchRP] = msc.rewardPoolFetch
	msc.RestHandlers[zmc.FetchBillingRatioRP] = msc.fetchBillingRatio
	msc.RestHandlers[zmc.UserRegisteredRP] = msc.userExist
	msc.RestHandlers[zmc.UserFetchRP] = msc.userFetch

	// metrics setup section
	msc.SmartContractExecutionStats[zmc.ConsumerRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.ConsumerRegisterFuncName, nil)
	msc.SmartContractExecutionStats[zmc.ProviderRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.ProviderRegisterFuncName, nil)
	msc.SmartContractExecutionStats[zmc.AccessPointRegisterFuncName] = metrics.GetOrRegisterCounter("sc:"+msc.ID+":func:"+zmc.AccessPointRegisterFuncName, nil)

	tests := [1]struct {
		name string
		want smartcontractinterface.SmartContractInterface
	}{
		{
			name: "OK",
			want: msc,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, want := NewMagmaSmartContract(), test.want.(*MagmaSmartContract)
			if fmt.Sprintf("%#v", got.SmartContract) != fmt.Sprintf("%#v", want.SmartContract) {
				t.Errorf("NewMagmaSmartContract() got: %#v | want: %#v", got.SmartContract, want.SmartContract)
			}
		})
	}
}

func Test_MagmaSmartContract_Execute(t *testing.T) {
	t.Parallel()

	msc, sci := mockSmartContractI(), mockStateContextI()
	cons, prov := mockConsumer(), mockProvider()

	tests := [8]struct {
		name  string
		txn   *tx.Transaction
		call  string
		blob  []byte
		sci   chain.StateContextI
		msc   *mockSmartContract
		error bool
	}{
		{
			name:  "Consumer_Register_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  zmc.ConsumerRegisterFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Consumer_Session_Start_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  zmc.ConsumerSessionStartFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Consumer_Session_Stop_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  zmc.ConsumerSessionStopFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Consumer_Update_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  zmc.ConsumerUpdateFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Provider_DataUsage_OK",
			txn:   &tx.Transaction{ClientID: prov.Id},
			call:  zmc.ProviderDataUsageFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Provider_Register_OK",
			txn:   &tx.Transaction{ClientID: prov.Id},
			call:  zmc.ProviderRegisterFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Provider_Update_OK",
			txn:   &tx.Transaction{ClientID: prov.Id},
			call:  zmc.ProviderUpdateFuncName,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Invalid_Func_Name_ERR",
			txn:   &tx.Transaction{ClientID: "not_present_id"},
			call:  "not_present_func_name",
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.Execute(test.txn, test.call, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("Execute() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && got != test.call {
				t.Errorf("Execute() got: %v | want: %v", got, test.call)
			}
		})
	}
}

func Test_MagmaSmartContract_GetAddress(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetAddress(); got != zmc.Address {
			t.Errorf("GetAddress() got: %v | want: %v", got, zmc.Address)
		}
	})
}

func Test_MagmaSmartContract_GetExecutionStats(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetExecutionStats(); !reflect.DeepEqual(got, msc.SmartContractExecutionStats) {
			t.Errorf("GetExecutionStats() got: %#v | want: %#v", got, msc.SmartContractExecutionStats)
		}
	})
}

func Test_MagmaSmartContract_GetHandlerStats(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()

	tests := [1]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  nil,
			msc:   msc,
			want:  "type string",
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.GetHandlerStats(test.ctx, test.vals)
			if (err != nil) != test.error {
				t.Errorf("GetHandlerStats() error: %v | want: %v", err, test.error)
				return
			}
			if _, ok := got.(string); !ok {
				t.Errorf("GetHandlerStats() got: %#v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_GetName(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetName(); got != Name {
			t.Errorf("GetName() got: %v | want: %v", got, Name)
		}
	})
}

func Test_MagmaSmartContract_GetRestPoints(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetRestPoints(); !reflect.DeepEqual(got, msc.RestHandlers) {
			t.Errorf("GetRestPoints() got: %#v | want: %#v", got, msc.RestHandlers)
		}
	})
}
