package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"
	"github.com/0chain/bandwidth_marketplace/code/core/time"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	"0chain.net/chaincore/tokenpool"
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
		ackn *tokenPool
		want []byte
	}{
		{
			name: "OK",
			ackn: pool,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.ackn.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_tokenPool_create(t *testing.T) {
	t.Parallel()

	ackn, sci := mockAcknowledgment(), mockStateContextI()
	amount, txn := ackn.Provider.Terms.GetAmount(), sci.GetTransaction()

	txn.Value = amount
	txn.ClientID = ackn.Consumer.ID

	resp := &tokenpool.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     ackn.SessionID,
		Value:      state.Balance(amount),
		FromClient: ackn.Consumer.ID,
		ToClient:   txn.ToClientID,
	}

	acknClientBalanceErr := mockAcknowledgment()
	acknClientBalanceErr.Consumer.ID = ""

	acknInsufficientFundsErr := mockAcknowledgment()
	acknInsufficientFundsErr.Consumer.ID = "insolvent_id"

	tests := [5]struct {
		name  string
		txn   *tx.Transaction
		ackn  *bmp.Acknowledgment
		pool  *tokenPool
		sci   chain.StateContextI
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   txn,
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			want:  string(resp.Encode()),
			error: false,
		},
		{
			name:  "Client_Balance_ERR",
			txn:   txn,
			ackn:  acknClientBalanceErr,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Insufficient_Funds_ERR",
			txn:   txn,
			ackn:  acknInsufficientFundsErr,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Add_Transfer_ERR",
			txn:   &tx.Transaction{ToClientID: "not_present_id"},
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
		{
			name:  "Insert_Token_Pool_ERR",
			txn:   &tx.Transaction{ToClientID: "cannot_insert_id"},
			ackn:  ackn,
			pool:  &tokenPool{},
			sci:   sci,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.pool.create(test.txn, test.ackn, test.sci)
			if err == nil && got != test.want {
				t.Errorf("create() got: %v | want: %v", got, test.want)
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

	pool, msc, sci := mockTokenPool(), mockMagmaSmartContract(), mockStateContextI()
	if _, err := sci.InsertTrieNode(pool.uid(msc.ID), pool); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	pool.ID = time.NowTime().String()
	if _, err := sci.InsertTrieNode(pool.uid(msc.ID), pool); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	poolInvalid := mockTokenPool()
	poolInvalid.ID = "not_present_id"

	txn := sci.GetTransaction()
	txnInvalid := txn.Clone()
	txnInvalid.ToClientID = "not_present_id"

	tests := [6]struct {
		name  string
		txn   *tx.Transaction
		bill  *bmp.Billing
		sci   chain.StateContextI
		pool  *tokenPool
		error bool
	}{
		{
			name:  "OK",
			txn:   txn,
			bill:  &bmp.Billing{Amount: int64(pool.Balance - pool.Balance/2)},
			sci:   sci,
			pool:  pool,
			error: false,
		},
		{
			name:  "Billing_Amount_Zero_Value_OK",
			txn:   txn,
			bill:  &bmp.Billing{Amount: 0},
			sci:   sci,
			pool:  mockTokenPool(),
			error: false,
		},
		{
			name:  "Billing_Amount_Negative_Value_ERR",
			txn:   txn,
			bill:  &bmp.Billing{Amount: -1},
			sci:   sci,
			pool:  mockTokenPool(),
			error: true,
		},
		{
			name:  "Transfer_Token_Pool_ERR",
			txn:   txnInvalid,
			bill:  &bmp.Billing{Amount: 1},
			sci:   sci,
			pool:  mockTokenPool(),
			error: true,
		},
		{
			name:  "Spend_Token_Pool_ERR",
			txn:   txnInvalid,
			bill:  &bmp.Billing{Amount: 1000},
			sci:   sci,
			pool:  mockTokenPool(),
			error: true,
		},
		{
			name:  "Delete_Trie_Node_ERR",
			txn:   txn,
			bill:  &bmp.Billing{Amount: 1000},
			sci:   sci,
			pool:  poolInvalid,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.pool.spend(test.txn, test.bill, test.sci); (err != nil) != test.error {
				t.Errorf("spend() error: %v | want: %v", err, test.error)
				t.Errorf("sci: %#v", sci.store)
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
