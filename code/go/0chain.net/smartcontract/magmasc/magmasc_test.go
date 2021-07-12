package magmasc

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
)

func Test_MagmaSmartContract_Execute(t *testing.T) {
	t.Parallel()

	msc, sci := mockSmartContractI(), mockStateContextI()
	blob, cons, prov := make([]byte, 0), mockConsumer(), mockProvider()

	tests := [7]struct {
		name  string
		txn   *tx.Transaction
		call  string
		blob  []byte
		sci   chain.StateContextI
		msc   *mockSmartContract
		error bool
	}{
		{
			name:  "Consumer_AcceptTerms_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  consumerAcceptTerms,
			blob:  blob,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Consumer_Register_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  consumerRegister,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Consumer_Session_Stop_OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			call:  consumerSessionStop,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Provider_DataUsage_OK",
			txn:   &tx.Transaction{ClientID: prov.ID},
			call:  providerDataUsage,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Provider_Register_OK",
			txn:   &tx.Transaction{ClientID: prov.ID},
			call:  providerRegister,
			blob:  nil,
			sci:   sci,
			msc:   msc,
			error: false,
		},
		{
			name:  "Provider_Terms_Update_OK",
			txn:   &tx.Transaction{ClientID: prov.ID},
			call:  providerTermsUpdate,
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

func TestMagmaSmartContract_GetAddress(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetAddress(); got != Address {
			t.Errorf("GetAddress() got: %v | want: %v", got, Address)
		}
	})
}

func TestMagmaSmartContract_GetExecutionStats(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := msc.GetExecutionStats(); !reflect.DeepEqual(got, msc.SmartContractExecutionStats) {
			t.Errorf("GetExecutionStats() got: %#v | want: %#v", got, msc.SmartContractExecutionStats)
		}
	})
}

func TestMagmaSmartContract_GetHandlerStats(t *testing.T) {
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
				t.Errorf("GetHandlerStats() error: %v, want: %v", err, test.error)
				return
			}
			if _, ok := got.(string); !ok {
				t.Errorf("GetHandlerStats() got: %#v | want: %v", got, test.want)
			}
		})
	}
}

func TestMagmaSmartContract_GetName(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		if got := msc.GetName(); got != Name {
			t.Errorf("GetName() got: %v | want: %v", got, Name)
		}
	})
}

func TestMagmaSmartContract_GetRestPoints(t *testing.T) {
	t.Parallel()

	msc := mockMagmaSmartContract()
	t.Run("OK", func(t *testing.T) {
		if got := msc.GetRestPoints(); !reflect.DeepEqual(got, msc.RestHandlers) {
			t.Errorf("GetRestPoints() got: %#v | want: %#v", got, msc.RestHandlers)
		}
	})
}
