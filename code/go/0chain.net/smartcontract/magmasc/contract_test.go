package magmasc

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"
	"github.com/0chain/bandwidth_marketplace/code/core/time"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/chaincore/state"
	tp "0chain.net/chaincore/tokenpool"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
)

func Test_MagmaSmartContract_acknowledgment(t *testing.T) {
	t.Parallel()

	msc, ackn, sci := mockMagmaSmartContract(), mockAcknowledgment(), mockStateContextI()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	acknInvalidJSON := bmp.Acknowledgment{SessionID: "invalid_json_id"}
	nodeInvalidJSON := mockInvalidJson{ID: acknInvalidJSON.SessionID}
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknInvalidJSON.SessionID, acknowledgment), &nodeInvalidJSON); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	acknInvalid := bmp.Acknowledgment{SessionID: "invalid_acknowledgment"}
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, acknInvalidJSON.SessionID, acknowledgment), &acknInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  *bmp.Acknowledgment
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
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

	msc, sci, list := mockMagmaSmartContract(), mockStateContextI(), mockConsumers()
	if _, err := sci.InsertTrieNode(AllConsumersKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sciInvalidJSON, node := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalidJSON.InsertTrieNode(AllConsumersKey, &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sorted := make([]*bmp.Consumer, len(list.Nodes.Sorted))
	copy(sorted, list.Nodes.Sorted)
	tests := [3]struct {
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
		{
			name:  "Not_Present_OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   mockStateContextI(),
			want:  Consumers{}.Nodes.Sorted,
			error: false,
		},
		{
			name:  "Decode_ERR",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sciInvalidJSON,
			want:  nil,
			error: true,
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

	msc, sci, list := mockMagmaSmartContract(), mockStateContextI(), mockProviders()
	if _, err := sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sciInvalidJSON, node := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalidJSON.InsertTrieNode(AllProvidersKey, &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sorted := make([]*bmp.Provider, len(list.Nodes.Sorted))
	copy(sorted, list.Nodes.Sorted)
	tests := [3]struct {
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
		{
			name:  "Not_Present_OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   mockStateContextI(),
			want:  Providers{}.Nodes.Sorted,
			error: false,
		},
		{
			name:  "Decode_ERR",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sciInvalidJSON,
			want:  nil,
			error: true,
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, cons.ExtID, consumerType), cons); err != nil {
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, cons.ExtID, consumerType), cons); err != nil {
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

	ackn, msc, sci := mockAcknowledgment(), mockMagmaSmartContract(), mockStateContextI()

	consList := Consumers{}
	if err := consList.add(msc.ID, ackn.Consumer, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, ackn.Provider, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	txn := &tx.Transaction{
		ClientID:   ackn.Consumer.ID,
		ToClientID: msc.ID,
	}

	req := &bmp.Acknowledgment{
		SessionID:     ackn.SessionID,
		AccessPointID: ackn.AccessPointID,
		Consumer:      &bmp.Consumer{ExtID: ackn.Consumer.ExtID},
		Provider:      &bmp.Provider{ExtID: ackn.Provider.ExtID},
	}

	resp := &tp.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		ToPool:     ackn.SessionID,
		Value:      state.Balance(ackn.Provider.Terms.GetAmount()),
		FromClient: txn.ClientID,
		ToClient:   txn.ToClientID,
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
			txn: &tx.Transaction{
				ClientID:   ackn.Consumer.ID,
				ToClientID: msc.ID,
			},
			blob:  req.Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(resp.Encode()),
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

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	ackn := mockAcknowledgment()
	ackn.SessionID += time.NowTime().String()
	ackn.Billing.CompletedAt = time.Now()
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	consList := Consumers{}
	if err := consList.add(msc.ID, ackn.Consumer, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, ackn.Provider, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	pool := tokenPool{PayerID: ackn.Consumer.ID, PayeeID: ackn.Provider.ID}
	pool.ID = ackn.SessionID
	pool.Balance = 1000
	if _, err := sci.InsertTrieNode(pool.uid(msc.ID), &pool); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	txn := &tx.Transaction{
		ClientID:   ackn.Consumer.ID,
		ToClientID: msc.ID,
	}

	resp := &tp.TokenPoolTransferResponse{
		TxnHash:    txn.Hash,
		FromPool:   pool.ID,
		FromClient: pool.PayerID,
		ToClient:   pool.PayeeID,
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
			name:  "OK",
			txn:   txn,
			blob:  (&bmp.Acknowledgment{SessionID: ackn.SessionID}).Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(resp.Encode()),
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
	if err := list.add(msc.ID, cons, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	cons = mockConsumer()
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
			txn:   &tx.Transaction{ClientID: cons.ID},
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, ackn.SessionID, acknowledgment), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	provList := Providers{}
	if err := provList.add(msc.ID, ackn.Provider, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	ackn.Billing.CalcAmount(ackn.Provider.Terms)
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
			txn:   &tx.Transaction{ClientID: ackn.Provider.ID},
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
				t.Errorf("blob: %v", string(test.blob))
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, prov.ExtID, providerType), prov); err != nil {
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
	if _, err := sci.InsertTrieNode(nodeUID(msc.ID, prov.ExtID, providerType), prov); err != nil {
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
			name:  "List_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_list"},
			blob:  nil,
			sci:   mockStateContextI(),
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

func Test_MagmaSmartContract_providerTerms(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov, provList := mockProvider(), Providers{}
	if err := provList.add(msc.ID, prov, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
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
			want:  prov.Terms,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"provider_id": {"not_present_id"}},
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

			got, err := test.msc.providerTerms(test.ctx, test.vals, test.sci)
			if (err != nil) != test.error {
				t.Errorf("providerTerms() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("providerTerms() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_providerUpdate(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	prov, list := mockProvider(), Providers{}
	if err := list.add(msc.ID, prov, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	prov = mockProvider()
	prov.Terms.Increase()
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
				t.Errorf("providerUpdate() error = %v, error %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("providerUpdate() got: %v, want: %v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_tokenPollFetch(t *testing.T) {
	t.Parallel()

	ackn, msc, sci := mockAcknowledgment(), mockMagmaSmartContract(), mockStateContextI()

	pool := tokenPool{
		PayerID: ackn.Consumer.ID,
		PayeeID: ackn.Provider.ID,
	}
	pool.ID = ackn.SessionID
	pool.Balance = 1000
	if _, err := sci.InsertTrieNode(pool.uid(msc.ID), &pool); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ackn  *bmp.Acknowledgment
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  *tokenPool
		error bool
	}{
		{
			name:  "OK",
			ackn:  ackn,
			sci:   sci,
			msc:   msc,
			want:  &pool,
			error: false,
		},
		{
			name:  "Value_Not_Present_ERR",
			ackn:  &bmp.Acknowledgment{SessionID: "not_present_id"},
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

			got, err := test.msc.tokenPollFetch(test.ackn, test.sci)
			if (err != nil) != test.error {
				t.Errorf("tokenPollFetch() error = %v, error %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("tokenPollFetch() got: %v, want: %v", got, test.want)
			}
		})
	}
}

func Test_nodeUID(t *testing.T) {
	t.Parallel()

	const (
		scID     = "sc_id"
		nodeID   = "node_id"
		nodeType = "node_type"
		wantUID  = "sc:" + scID + colon + nodeType + colon + nodeID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		if got := nodeUID(scID, nodeID, nodeType); got != wantUID {
			t.Errorf("nodeUID() got: %v | want: %v", got, wantUID)
		}
	})
}
