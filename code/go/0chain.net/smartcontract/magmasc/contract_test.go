package magmasc

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"
	"github.com/0chain/gosdk/zmagmacore/time"

	chain "0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
	store "0chain.net/core/ememorystore"
)

func Test_MagmaSmartContract_acknowledgment(t *testing.T) {
	t.Parallel()

	msc, ackn, sci := mockMagmaSmartContract(), mockAcknowledgment(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	acknInvalidJSON := zmc.Acknowledgment{SessionID: "invalid_json_id"}
	nodeInvalidJSON := mockInvalidJson{ID: acknInvalidJSON.SessionID}
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, acknInvalidJSON.SessionID), &nodeInvalidJSON); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	acknInvalid := zmc.Acknowledgment{SessionID: "invalid_acknowledgment"}
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, acknInvalidJSON.SessionID), &acknInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		id    string
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  *zmc.Acknowledgment
		error bool
	}{
		{
			name:  "OK",
			id:    ackn.SessionID,
			sci:   sci,
			msc:   msc,
			want:  ackn,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			id:    "not_present_id",
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
		{
			name:  "Decode_ERR",
			id:    nodeInvalidJSON.ID,
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
		{
			name:  "Invalid_ERR",
			id:    acknInvalid.SessionID,
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgment(test.id, test.sci)
			if (err != nil) != test.error {
				t.Errorf("acknowledgment() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("acknowledgment() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_acknowledgmentAccepted(t *testing.T) {
	t.Parallel()

	msc, ackn, sci := mockMagmaSmartContract(), mockAcknowledgment(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"id": {ackn.SessionID}},
			sci:   sci,
			msc:   msc,
			want:  ackn,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgmentAccepted(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("acknowledgmentAccepted() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("acknowledgmentAccepted() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_acknowledgmentAcceptedVerify(t *testing.T) {
	t.Parallel()

	msc, ackn, sci := mockMagmaSmartContract(), mockAcknowledgment(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [5]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name: "OK",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"access_point_id": {ackn.AccessPointID},
				"consumer_ext_id": {ackn.Consumer.ExtID},
				"provider_ext_id": {ackn.Provider.ExtID},
			},
			sci:   sci,
			msc:   msc,
			want:  ackn,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"session_id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
		{
			name: "Invalid_Access_Point_ERR",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"consumer_ext_id": {ackn.Consumer.ExtID},
				"provider_ext_id": {ackn.Provider.ExtID},
			},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
		{
			name: "Invalid_Consumer_ExtID_ERR",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"access_point_id": {ackn.AccessPointID},
				"provider_ext_id": {ackn.Provider.ExtID},
			},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
		{
			name: "Invalid_Provider_ExtID_ERR",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"access_point_id": {ackn.AccessPointID},
				"consumer_ext_id": {ackn.Consumer.ExtID},
			},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgmentAcceptedVerify(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("acknowledgmentAcceptedVerify() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("acknowledgmentAcceptedVerify() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_acknowledgmentExist(t *testing.T) {
	t.Parallel()

	msc, ackn, sci := mockMagmaSmartContract(), mockAcknowledgment(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"id": {ackn.SessionID}},
			sci:   sci,
			msc:   msc,
			want:  true,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  false,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgmentExist(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("acknowledgmentExist() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("acknowledgmentExist() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_allConsumers(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockConsumers(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockConsumer(), store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	sorted := make([]*zmc.Consumer, len(list.Sorted))
	copy(sorted, list.Sorted)

	tests := [1]struct {
		name  string
		msc   *MagmaSmartContract
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sci,
			want:  sorted,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.allConsumers(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("allConsumers() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("allConsumers() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_allProviders(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockProviders(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockProvider(), store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	sorted := make([]*zmc.Provider, len(list.Sorted))
	copy(sorted, list.Sorted)

	tests := [1]struct {
		name  string
		msc   *MagmaSmartContract
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   nil,
			want:  sorted,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.allProviders(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("allProviders() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("allProviders() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerExist(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	cons := mockConsumer()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, consumerType, cons.ExtID), cons); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"ext_id": {cons.ExtID}},
			sci:   sci,
			msc:   msc,
			want:  true,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"ext_id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  false,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.consumerExist(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerExist() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("consumerExist() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerFetch(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	cons := mockConsumer()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, consumerType, cons.ExtID), cons); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"ext_id": {cons.ExtID}},
			sci:   sci,
			msc:   msc,
			want:  cons,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.consumerFetch(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerFetch() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("consumerFetch() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerRegister(t *testing.T) {
	t.Parallel()

	cons, msc, sci := mockConsumer(), mockMagmaSmartContract(), mockStateContextI()
	blob := cons.Encode()

	sciInvalid, nodeInvalid := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalid.InsertTrieNode(AllConsumersKey, &nodeInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   &tx.Transaction{ClientID: cons.ID},
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  string(blob),
			error: false,
		},
		{
			name:  "Extract_Consumers_ERR",
			txn:   nil,
			blob:  nil,
			sci:   sciInvalid,
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "List_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_list"},
			blob:  nil,
			sci:   mockStateContextI(),
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "Consumer_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_id"},
			blob:  nil,
			sci:   mockStateContextI(),
			msc:   msc,
			want:  "",
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.consumerRegister(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerRegister() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("consumerRegister() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerSessionStart(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	ackn := mockAcknowledgment()
	ackn.Billing = zmc.Billing{} // initial value
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = ackn.Consumer.ID

	pool := newTokenPool()
	if err := pool.create(txn, ackn, sci); err != nil {
		t.Fatalf("tokenPool.create() error: %v | want: %v", err, nil)
	}
	ackn.TokenPool = &pool.TokenPool

	consList := Consumers{}
	if err := consList.add(msc.ID, ackn.Consumer, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, ackn.Provider, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name: "OK",
			txn:  txn,
			blob: (&zmc.Acknowledgment{
				SessionID:     ackn.SessionID,
				AccessPointID: ackn.AccessPointID,
				Consumer:      &zmc.Consumer{ExtID: ackn.Consumer.ExtID},
				Provider:      &zmc.Provider{ExtID: ackn.Provider.ExtID},
			}).Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(ackn.Encode()),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.consumerSessionStart(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerSessionStart() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("consumerSessionStart() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerSessionStop(t *testing.T) {
	t.Parallel()

	ackn, msc, sci := mockAcknowledgment(), mockMagmaSmartContract(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	consList := Consumers{}
	if err := consList.add(msc.ID, ackn.Consumer, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, ackn.Provider, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = ackn.Consumer.ID

	pool := newTokenPool()
	pool.PayerID = ackn.Consumer.ID
	pool.PayeeID = ackn.Provider.ID
	pool.ID = ackn.SessionID
	pool.Balance = 1000
	pool.Transfers = []zmc.TokenPoolTransfer{{
		TxnHash:    txn.Hash,
		FromPool:   pool.ID,
		FromClient: pool.PayerID,
		ToClient:   pool.PayeeID,
	}}

	ackn.Billing.CompletedAt = time.Now()
	ackn.TokenPool = &pool.TokenPool

	tests := [1]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name: "OK",
			txn:  txn,
			blob: (&zmc.Acknowledgment{
				SessionID:     ackn.SessionID,
				AccessPointID: ackn.AccessPointID,
				Consumer:      &zmc.Consumer{ExtID: ackn.Consumer.ExtID},
				Provider:      &zmc.Provider{ExtID: ackn.Provider.ExtID},
			}).Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(ackn.Encode()),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.consumerSessionStop(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerSessionStop() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("consumerSessionStop() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerUpdate(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()
	cons, list := mockConsumer(), Consumers{}
	if err := list.add(msc.ID, cons, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = cons.ID
	blob := cons.Encode()

	tests := [1]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   txn,
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  string(blob),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.consumerUpdate(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerUpdate() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("consumerUpdate() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_providerDataUsage(t *testing.T) {
	t.Parallel()

	ackn, msc, sci := mockAcknowledgment(), mockMagmaSmartContract(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknowledgment, ackn.SessionID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	list := Providers{}
	if err := list.add(msc.ID, ackn.Provider, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = ackn.Provider.ID

	pool := newTokenPool()
	pool.PayerID = ackn.Consumer.ID
	pool.PayeeID = ackn.Provider.ID
	pool.ID = ackn.SessionID
	pool.Balance = 1000
	pool.Transfers = []zmc.TokenPoolTransfer{{
		TxnHash:    txn.Hash,
		FromPool:   pool.ID,
		FromClient: pool.PayerID,
		ToClient:   pool.PayeeID,
	}}

	ackn.Billing.CalcAmount(ackn.Terms)
	ackn.TokenPool = &pool.TokenPool

	tests := [1]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   txn,
			blob:  ackn.Billing.DataUsage.Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(ackn.Encode()),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.providerDataUsage(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerDataUsage() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("providerDataUsage() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_providerExist(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov := mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, providerType, prov.ExtID), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"ext_id": {prov.ExtID}},
			sci:   sci,
			msc:   msc,
			want:  true,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"ext_id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  false,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.providerExist(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerExist() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("providerExist() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_providerFetch(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov := mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, providerType, prov.ExtID), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error bool
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"ext_id": {prov.ExtID}},
			sci:   sci,
			msc:   msc,
			want:  prov,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.providerFetch(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerFetch() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("providerFetch() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_providerRegister(t *testing.T) {
	t.Parallel()

	prov, msc, sci := mockProvider(), mockMagmaSmartContract(), mockStateContextI()
	blob := prov.Encode()

	sciInvalid, nodeInvalid := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalid.InsertTrieNode(AllProvidersKey, &nodeInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   &tx.Transaction{ClientID: prov.ID},
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  string(blob),
			error: false,
		},
		{
			name:  "Extract_Providers_ERR",
			txn:   nil,
			blob:  nil,
			sci:   sciInvalid,
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "Provider_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_id"},
			blob:  nil,
			sci:   mockStateContextI(),
			msc:   msc,
			want:  "",
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.providerRegister(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerRegister() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("providerRegister() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func TestMagmaSmartContract_providerSessionInit(t *testing.T) {
	t.Parallel()

	ackn, msc, sci := mockAcknowledgment(), mockMagmaSmartContract(), mockStateContextI()

	ackn.Billing = zmc.Billing{}
	blob := ackn.Encode()

	consList := Consumers{}
	if err := consList.add(msc.ID, ackn.Consumer, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, ackn.Provider, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = ackn.Provider.ID

	tests := [1]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   txn,
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  string(blob),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.providerSessionInit(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerSessionInit() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("providerSessionInit() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_providerUpdate(t *testing.T) {
	t.Parallel()

	list, msc, sci := Providers{}, mockMagmaSmartContract(), mockStateContextI()

	prov := mockProvider()
	if err := list.add(msc.ID, prov, store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	prov.Host = "update.provider.host.local"
	blob := prov.Encode()

	tests := [1]struct {
		name  string
		txn   *tx.Transaction
		blob  []byte
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  string
		error bool
	}{
		{
			name:  "OK",
			txn:   &tx.Transaction{ClientID: prov.ID},
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  string(blob),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.providerUpdate(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerUpdate() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("providerUpdate() got: %v, want: %v", got, test.want)
			}
		})
	}
}

func Test_nodeUID(t *testing.T) {
	t.Parallel()

	const (
		nodeID   = "id:node"
		nodeType = "type:node"
		wantUID  = "sc:" + Address + colon + nodeType + colon + nodeID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := nodeUID(Address, nodeType, nodeID); got != wantUID {
			t.Errorf("nodeUID() got: %v | want: %v", got, wantUID)
		}
	})
}
