package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
)

func Test_tokenPool_Decode(t *testing.T) {
	t.Parallel()

	pool := mockTokenPool()
	blob, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		blob  []byte
		want  *tokenPool
		error bool
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  pool,
			error: false,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &tokenPool{},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &tokenPool{}
			if err := got.Decode(test.blob); (err != nil) != test.error {
				t.Errorf("Decode() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_tokenPool_Encode(t *testing.T) {
	t.Parallel()

	pool := mockTokenPool()
	blob, err := json.Marshal(pool)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		pool *tokenPool
		want []byte
	}{
		{
			name: "OK",
			pool: pool,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.pool.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_tokenPool_create(t *testing.T) {
	t.Parallel()

	ackn, sci := mockAcknowledgment(), mockStateContextI()

	terms := ackn.Provider.Terms[ackn.AccessPointID]
	amount := terms.GetAmount()

	txn := sci.GetTransaction()
	txn.Value = amount
	txn.ClientID = ackn.Consumer.ID

	acknClientBalanceErr := mockAcknowledgment()
	acknClientBalanceErr.Consumer.ID = ""

	acknInsufficientFundsErr := mockAcknowledgment()
	acknInsufficientFundsErr.Consumer.ID = "insolvent_id"

	tests := [4]struct {
		name  string
		txn   *tx.Transaction
		ackn  *zmc.Acknowledgment
		pool  *tokenPool
		sci   chain.StateContextI
		want  *zmc.TokenPoolTransfer
		error bool
	}{
		{
			name: "OK",
			txn:  txn,
			ackn: ackn,
			pool: &tokenPool{},
			sci:  sci,
			want: &zmc.TokenPoolTransfer{
				TxnHash:    txn.Hash,
				ToPool:     ackn.SessionID,
				Value:      amount,
				FromClient: ackn.Consumer.ID,
				ToClient:   txn.ToClientID,
			},
			error: false,
		},
		{
			name:  "Client_Balance_ERR",
			txn:   txn,
			ackn:  acknClientBalanceErr,
			pool:  &tokenPool{},
			sci:   sci,
			want:  nil,
			error: true,
		},
		{
			name:  "Insufficient_Funds_ERR",
			txn:   txn,
			ackn:  acknInsufficientFundsErr,
			pool:  &tokenPool{},
			sci:   sci,
			want:  nil,
			error: true,
		},
		{
			name:  "Add_Transfer_ERR",
			txn:   &tx.Transaction{ToClientID: "not_present_id"},
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.pool.create(test.txn, test.ackn, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("create() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("create() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_tokenPool_spend(t *testing.T) {
	t.Parallel()

	sci := mockStateContextI()
	txn := sci.GetTransaction()
	txnInvalid := txn.Clone()
	txnInvalid.ToClientID = "not_present_id"

	poolOK1, poolOK2 := mockTokenPool(), mockTokenPool()
	tests := [5]struct {
		name  string
		txn   *tx.Transaction
		bill  *zmc.Billing
		sci   chain.StateContextI
		pool  *tokenPool
		want  *zmc.TokenPoolTransfer
		error bool
	}{
		{
			name: "OK",
			txn:  txn,
			bill: &zmc.Billing{Amount: int64(poolOK1.Balance - poolOK1.Balance/2)},
			sci:  sci,
			pool: poolOK1,
			want: &zmc.TokenPoolTransfer{
				TxnHash:    txn.Hash,
				FromPool:   poolOK1.ID,
				Value:      poolOK1.Balance - poolOK1.Balance/2,
				FromClient: poolOK1.PayerID,
				ToClient:   poolOK1.PayeeID,
			},
			error: false,
		},
		{
			name: "Billing_Amount_Zero_OK",
			txn:  txn,
			bill: &zmc.Billing{Amount: 0},
			sci:  sci,
			pool: poolOK2,
			want: &zmc.TokenPoolTransfer{
				TxnHash:    txn.Hash,
				FromPool:   poolOK2.ID,
				Value:      0,
				FromClient: poolOK2.PayerID,
				ToClient:   poolOK2.PayeeID,
			},
			error: false,
		},
		{
			name:  "Billing_Amount_Negative_Value_ERR",
			txn:   txn,
			bill:  &zmc.Billing{Amount: -1},
			sci:   sci,
			pool:  mockTokenPool(),
			want:  nil,
			error: true,
		},
		{
			name:  "Transfer_Token_Pool_ERR",
			txn:   txnInvalid,
			bill:  &zmc.Billing{Amount: 1},
			sci:   sci,
			pool:  mockTokenPool(),
			want:  nil,
			error: true,
		},
		{
			name:  "Spend_Token_Pool_ERR",
			txn:   txnInvalid,
			bill:  &zmc.Billing{Amount: 1000},
			sci:   sci,
			pool:  mockTokenPool(),
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.pool.spend(test.txn, test.bill, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("create() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("spend() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_tokenPool_uid(t *testing.T) {
	t.Parallel()

	const (
		scID  = "sc_uid"
		tpID  = "token_pool_id"
		tpUID = "sc:" + scID + ":tokenpool:" + tpID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		pool := tokenPool{}
		pool.ID = tpID

		if got := pool.uid(scID); got != tpUID {
			t.Errorf("uid() got: %v | want: %v", got, tpUID)
		}
	})
}
