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
)

func Test_MagmaSmartContract_session(t *testing.T) {
	t.Parallel()

	msc, sess, sci := mockMagmaSmartContract(), mockSession(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sessInvalidJSON := zmc.Session{SessionID: "invalid_json_id"}
	nodeInvalidJSON := mockInvalidJson{ID: sessInvalidJSON.SessionID}
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sessInvalidJSON.SessionID), &nodeInvalidJSON); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sessInvalid := zmc.Session{SessionID: "invalid_session"}
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sessInvalidJSON.SessionID), &sessInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		id    string
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  *zmc.Session
		error bool
	}{
		{
			name:  "OK",
			id:    sess.SessionID,
			sci:   sci,
			msc:   msc,
			want:  sess,
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
			id:    sessInvalid.SessionID,
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

			got, err := test.msc.session(test.id, test.sci)
			if (err != nil) != test.error {
				t.Errorf("session() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("session() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_sessionAccepted(t *testing.T) {
	t.Parallel()

	msc, sess, sci := mockMagmaSmartContract(), mockSession(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
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
			vals:  url.Values{"id": {sess.SessionID}},
			sci:   sci,
			msc:   msc,
			want:  sess,
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

			got, err := test.msc.sessionAccepted(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("sessionAccepted() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("sessionAccepted() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_sessionAcceptedVerify(t *testing.T) {
	t.Parallel()

	msc, sess, sci := mockMagmaSmartContract(), mockSession(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
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
				"session_id":      {sess.SessionID},
				"access_point_id": {sess.AccessPoint.ID},
				"consumer_ext_id": {sess.Consumer.ExtID},
				"provider_ext_id": {sess.Provider.ExtID},
			},
			sci:   sci,
			msc:   msc,
			want:  sess,
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
				"session_id":      {sess.SessionID},
				"consumer_ext_id": {sess.Consumer.ExtID},
				"provider_ext_id": {sess.Provider.ExtID},
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
				"session_id":      {sess.SessionID},
				"access_point_id": {sess.AccessPoint.ID},
				"provider_ext_id": {sess.Provider.ExtID},
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
				"session_id":      {sess.SessionID},
				"access_point_id": {sess.AccessPoint.ID},
				"consumer_ext_id": {sess.Consumer.ExtID},
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

			got, err := test.msc.sessionAcceptedVerify(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("sessionAcceptedVerify() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("sessionAcceptedVerify() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_sessionExist(t *testing.T) {
	t.Parallel()

	msc, sess, sci := mockMagmaSmartContract(), mockSession(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
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
			vals:  url.Values{"id": {sess.SessionID}},
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

			got, err := test.msc.sessionExist(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("sessionExist() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("sessionExist() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_allConsumers(t *testing.T) {
	t.Parallel()

	list, msc, sci := mockConsumers(), mockMagmaSmartContract(), mockStateContextI()
	if err := list.add(msc.ID, mockConsumer(), msc.db, sci); err != nil {
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
	if err := list.add(msc.ID, mockProvider(), msc.db, sci); err != nil {
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

	sess := mockSession()
	sess.Billing = zmc.Billing{} // initial value
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = sess.Consumer.ID

	pool := newTokenPool()
	if err := pool.create(txn, sess, sci); err != nil {
		t.Fatalf("tokenPool.create() error: %v | want: %v", err, nil)
	}
	sess.TokenPool = &pool.TokenPool

	consList := Consumers{}
	if err := consList.add(msc.ID, sess.Consumer, msc.db, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, sess.Provider, msc.db, sci); err != nil {
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
			blob: (&zmc.Session{
				SessionID:   sess.SessionID,
				AccessPoint: sess.AccessPoint,
				Consumer:    &zmc.Consumer{ExtID: sess.Consumer.ExtID},
				Provider:    &zmc.Provider{ExtID: sess.Provider.ExtID},
			}).Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(sess.Encode()),
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

	sess, msc, sci := mockSession(), mockMagmaSmartContract(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	consList := Consumers{}
	if err := consList.add(msc.ID, sess.Consumer, msc.db, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, sess.Provider, msc.db, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = sess.Consumer.ID

	pool := newTokenPool()
	pool.PayerID = sess.Consumer.ID
	pool.PayeeID = sess.Provider.ID
	pool.ID = sess.SessionID
	pool.Balance = 1000
	pool.Transfers = []zmc.TokenPoolTransfer{{
		TxnHash:    txn.Hash,
		FromPool:   pool.ID,
		FromClient: pool.PayerID,
		ToClient:   pool.PayeeID,
	}}

	sess.Billing.CompletedAt = time.Now()
	sess.TokenPool = &pool.TokenPool

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
			blob: (&zmc.Session{
				SessionID:   sess.SessionID,
				AccessPoint: sess.AccessPoint,
				Consumer:    &zmc.Consumer{ExtID: sess.Consumer.ExtID},
				Provider:    &zmc.Provider{ExtID: sess.Provider.ExtID},
			}).Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(sess.Encode()),
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
	if err := list.add(msc.ID, cons, msc.db, sci); err != nil {
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

	sess, msc, sci := mockSession(), mockMagmaSmartContract(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, session, sess.SessionID), sess); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	list := Providers{}
	if err := list.add(msc.ID, sess.Provider, msc.db, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = sess.Provider.ID

	pool := newTokenPool()
	pool.PayerID = sess.Consumer.ID
	pool.PayeeID = sess.Provider.ID
	pool.ID = sess.SessionID
	pool.Balance = 1000
	pool.Transfers = []zmc.TokenPoolTransfer{{
		TxnHash:    txn.Hash,
		FromPool:   pool.ID,
		FromClient: pool.PayerID,
		ToClient:   pool.PayeeID,
	}}

	sess.Billing.CalcAmount(sess.AccessPoint.Terms)
	sess.TokenPool = &pool.TokenPool

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
			blob:  sess.Billing.DataUsage.Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(sess.Encode()),
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

	sess, msc, sci := mockSession(), mockMagmaSmartContract(), mockStateContextI()
	sess.Provider.MinStake = 1
	sess.AccessPoint.MinStake = 1

	sess.Billing = zmc.Billing{}
	blob := sess.Encode()

	consList := Consumers{}
	if err := consList.add(msc.ID, sess.Consumer, msc.db, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, sess.Provider, msc.db, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	apList := AccessPoints{}
	if err := apList.add(msc.ID, sess.AccessPoint, msc.db, sci); err != nil {
		t.Fatalf("AccessPoints.add() error: %v | want: %v", err, nil)
	}

	txn := sci.GetTransaction()
	txn.ClientID = sess.Provider.ID

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
	if err := list.add(msc.ID, prov, msc.db, sci); err != nil {
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

func Test_MagmaSmartContract_accessPointRegister(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov := mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, providerType, prov.ExtID), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	ap := mockAccessPoint(prov.ExtID)
	blob := ap.Encode()

	sciInvalid, nodeInvalid := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalid.InsertTrieNode(AllAccessPointsKey, &nodeInvalid); err != nil {
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
			txn:   &tx.Transaction{ClientID: ap.ID},
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  string(blob),
			error: false,
		},
		{
			name:  "Extract_AccessPoints_ERR",
			txn:   nil,
			blob:  nil,
			sci:   sciInvalid,
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "AccessPoint_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_id"},
			blob:  nil,
			sci:   mockStateContextI(),
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "Provider_Is_Not_registered_ERR",
			txn:   &tx.Transaction{ClientID: ap.ID},
			blob:  mockAccessPoint("invalid_provider").Encode(),
			sci:   sci,
			msc:   msc,
			want:  "",
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.accessPointRegister(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("accessPointRegister() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("accessPointRegister() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_accessPointUpdate(t *testing.T) {
	t.Parallel()

	list, msc, sci := AccessPoints{}, mockMagmaSmartContract(), mockStateContextI()

	prov := mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, providerType, prov.ExtID), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	ap := mockAccessPoint(prov.ExtID)
	if err := list.add(msc.ID, ap, msc.db, sci); err != nil {
		t.Fatalf("add() error: %v | want: %v", err, nil)
	}

	ap.Terms.Price++

	blob := ap.Encode()

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
			txn:   &tx.Transaction{ClientID: ap.ID},
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

			got, err := test.msc.accessPointUpdate(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("accessPointUpdate() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("accessPointUpdate() got: %v, want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_accessPointExist(t *testing.T) {
	t.Parallel()

	ap, msc, sci := mockAccessPoint("prov-ext-id"), mockMagmaSmartContract(), mockStateContextI()

	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, accessPointType, ap.ID), ap); err != nil {
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
			vals:  url.Values{"id": {ap.ID}},
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

			got, err := test.msc.accessPointExist(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("accessPointExist() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("accessPointExist() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_accessPointFetch(t *testing.T) {
	t.Parallel()

	ap, msc, sci := mockAccessPoint("prov-ext-id"), mockMagmaSmartContract(), mockStateContextI()

	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, accessPointType, ap.ID), ap); err != nil {
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
			vals:  url.Values{"id": {ap.ID}},
			sci:   sci,
			msc:   msc,
			want:  ap,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.accessPointFetch(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("accessPointFetch() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("accessPointFetch() got: %v | want: %v", got, test.want)
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
