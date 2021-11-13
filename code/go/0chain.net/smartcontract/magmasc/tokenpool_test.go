package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/magmasc/pb"
	"github.com/stretchr/testify/assert"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
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
			want:  newTokenPool(),
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := newTokenPool()
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

	sess, sci := mockSession(), mockStateContextI()
	sess.Billing.Ratio = 1

	txn := sci.GetTransaction()
	txn.Value = sess.AccessPoint.TermsGetAmount()
	txn.ClientID = sess.Consumer.ID

	sessClientBalanceErr := mockSession()
	sessClientBalanceErr.Consumer.ID = ""

	sessInsufficientFundsErr := mockSession()
	sessInsufficientFundsErr.Billing.Ratio = 1
	sessInsufficientFundsErr.Consumer.ID = "insolvent_id"

	tests := [3]struct {
		name  string
		txn   *tx.Transaction
		sess  *zmc.Session
		pool  *tokenPool
		sci   chain.StateContextI
		want  []*pb.TokenPoolTransfer
		error bool
	}{
		{
			name: "OK",
			txn:  txn,
			sess: sess,
			pool: newTokenPool(),
			sci:  sci,
			want: []*pb.TokenPoolTransfer{{
				TxnHash:    txn.Hash,
				ToPool:     sess.SessionID,
				Value:      sess.AccessPoint.TermsGetAmount(),
				FromClient: sess.Consumer.ID,
				ToClient:   txn.ToClientID,
			}},
			error: false,
		},
		{
			name:  "Client_Balance_ERR",
			txn:   txn,
			sess:  sessClientBalanceErr,
			pool:  newTokenPool(),
			sci:   sci,
			want:  newTokenPool().Transfers,
			error: true,
		},
		{
			name:  "Insufficient_Funds_ERR",
			txn:   txn,
			sess:  sessInsufficientFundsErr,
			pool:  newTokenPool(),
			sci:   sci,
			want:  newTokenPool().Transfers,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.pool.create(test.txn, test.sess, test.sci); (err != nil) != test.error {
				t.Errorf("create() error: %v | want: %v", err, test.error)
				return
			}

			assert.Equal(t, test.want, test.pool.Transfers, "create()")
		})
	}
}

func Test_tokenPool_spend(t *testing.T) {
	t.Parallel()

	sci := mockStateContextI()
	txn := sci.GetTransaction()

	poolOK1, poolOK2 := mockTokenPool(), mockTokenPool()
	poolInvalid := mockTokenPool()
	poolInvalid.HolderId = "not_present_id"
	tests := [5]struct {
		name   string
		txn    *tx.Transaction
		amount state.Balance
		sci    chain.StateContextI
		pool   *tokenPool
		want   []*pb.TokenPoolTransfer
		error  bool
	}{
		{
			name:   "OK",
			txn:    txn,
			amount: state.Balance(poolOK1.Balance - poolOK1.Balance/2),
			sci:    sci,
			pool:   poolOK1,
			want: []*pb.TokenPoolTransfer{
				{
					TxnHash:    txn.Hash,
					FromPool:   poolOK1.Id,
					Value:      poolOK1.Balance - poolOK1.Balance/2,
					FromClient: poolOK1.PayerId,
					ToClient:   poolOK1.PayeeId,
				},
				{
					TxnHash:  txn.Hash,
					FromPool: poolOK1.Id,
					Value:    poolOK1.Balance - poolOK1.Balance/2,
					ToClient: poolOK1.PayerId,
				},
			},
			error: false,
		},
		{
			name:   "Billing_Amount_Zero_OK",
			txn:    txn,
			amount: 0,
			sci:    sci,
			pool:   poolOK2,
			want: []*pb.TokenPoolTransfer{{
				TxnHash:  txn.Hash,
				FromPool: poolOK2.Id,
				Value:    1000,
				ToClient: poolOK2.PayerId,
			}},
			error: false,
		},
		{
			name:   "Billing_Amount_Negative_Value_ERR",
			txn:    txn,
			amount: -1,
			sci:    sci,
			pool:   mockTokenPool(),
			want:   mockTokenPool().Transfers,
			error:  true,
		},
		{
			name:   "Transfer_Token_Pool_ERR",
			txn:    txn,
			amount: 1,
			sci:    sci,
			pool:   poolInvalid,
			want:   poolInvalid.Transfers,
			error:  true,
		},
		{
			name:   "Spend_Token_Pool_ERR",
			txn:    txn,
			amount: 1000,
			sci:    sci,
			pool:   poolInvalid,
			want:   poolInvalid.Transfers,
			error:  true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.pool.spend(test.txn, test.amount, test.sci); (err != nil) != test.error {
				t.Errorf("spend() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(test.pool.Transfers, test.want) {
				t.Errorf("create() got: %#v | want: %#v", test.pool.Transfers, test.want)
				for _, trans := range test.pool.Transfers {
					t.Errorf("create() got: %#v", trans)
				}
				for _, trans := range test.want {
					t.Errorf("create() want: %#v", trans)
				}
			}
		})
	}
}
