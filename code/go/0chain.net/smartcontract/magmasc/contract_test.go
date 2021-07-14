package magmasc

import (
	"context"
	"net/url"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	tx "0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func Test_MagmaSmartContract_acknowledgment(t *testing.T) {
	t.Parallel()

	msc, ackn, sci := mockMagmaSmartContract(), mockAcknowledgment(), mockStateContextI()
	if _, err := sci.InsertTrieNode(ackn.uid(msc.ID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	acknInvalidJSON := Acknowledgment{SessionID: "invalid_json_id"}
	nodeInvalidJSON := mockInvalidJson{ID: acknInvalidJSON.SessionID}
	if _, err := sci.InsertTrieNode(acknInvalidJSON.uid(msc.ID), &nodeInvalidJSON); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	acknInvalid := Acknowledgment{SessionID: "invalid_acknowledgment"}
	if _, err := sci.InsertTrieNode(acknInvalid.uid(msc.ID), &acknInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  *Acknowledgment
		error error
	}{
		{
			name:  "OK",
			id:    ackn.SessionID,
			sci:   sci,
			msc:   msc,
			want:  ackn,
			error: nil,
		},
		{
			name:  "Not_Present_ERR",
			id:    "not_present_id",
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: util.ErrValueNotPresent,
		},
		{
			name:  "Decode_ERR",
			id:    nodeInvalidJSON.ID,
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: errDecodeData,
		},
		{
			name:  "Invalid_ERR",
			id:    acknInvalid.SessionID,
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: errAcknowledgmentInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgment(test.id, test.sci)
			if !errIs(err, test.error) {
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
	if _, err := sci.InsertTrieNode(ackn.uid(msc.ID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error error
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"id": {ackn.SessionID}},
			sci:   sci,
			msc:   msc,
			want:  ackn,
			error: nil,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: util.ErrValueNotPresent,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgmentAccepted(test.ctx, test.vals, test.sci)
			if !errIs(err, test.error) {
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
	if _, err := sci.InsertTrieNode(ackn.uid(msc.ID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [5]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error error
	}{
		{
			name: "OK",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"access_point_id": {ackn.AccessPointID},
				"consumer_id":     {ackn.Consumer.ID},
				"provider_id":     {ackn.Provider.ID},
			},
			sci:   sci,
			msc:   msc,
			want:  ackn,
			error: nil,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"session_id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: util.ErrValueNotPresent,
		},
		{
			name: "Verify_Access_Point_ERR",
			ctx:  nil,
			vals: url.Values{
				"session_id":  {ackn.SessionID},
				"consumer_id": {ackn.Consumer.ID},
				"provider_id": {ackn.Provider.ID},
			},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: errVerifyAccessPointID,
		},
		{
			name: "Verify_Consumer_ERR",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"access_point_id": {ackn.AccessPointID},
				"provider_id":     {ackn.Provider.ID},
			},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: errVerifyConsumerID,
		},
		{
			name: "Verify_Provider_ERR",
			ctx:  nil,
			vals: url.Values{
				"session_id":      {ackn.SessionID},
				"access_point_id": {ackn.AccessPointID},
				"consumer_id":     {ackn.Consumer.ID},
			},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: errVerifyProviderID,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgmentAcceptedVerify(test.ctx, test.vals, test.sci)
			if !errIs(err, test.error) {
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
	if _, err := sci.InsertTrieNode(ackn.uid(msc.ID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error error
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"id": {ackn.SessionID}},
			sci:   sci,
			msc:   msc,
			want:  true,
			error: nil,
		},
		{
			name:  "Not_Present_ERR",
			ctx:   nil,
			vals:  url.Values{"id": {"not_present_id"}},
			sci:   sci,
			msc:   msc,
			want:  false,
			error: nil,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.acknowledgmentExist(test.ctx, test.vals, test.sci)
			if !errIs(err, test.error) {
				t.Errorf("acknowledgmentAccepted() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("acknowledgmentAccepted() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_allConsumers(t *testing.T) {
	t.Parallel()

	msc, sci, list := mockMagmaSmartContract(), mockStateContextI(), mockConsumers()
	if _, err := sci.InsertTrieNode(AllConsumersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	sciInvalidJSON, node := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalidJSON.InsertTrieNode(AllConsumersKey, &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		msc   *MagmaSmartContract
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		want  interface{}
		error error
	}{
		{
			name:  "OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sci,
			want:  list.Nodes.Sorted,
			error: nil,
		},
		{
			name:  "Not_Present_OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   mockStateContextI(),
			want:  Consumers{Nodes: &consumersSorted{}}.Nodes.Sorted,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sciInvalidJSON,
			want:  nil,
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.allConsumers(test.ctx, test.vals, test.sci)
			if !errIs(err, test.error) {
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
	if _, err := sci.InsertTrieNode(AllProvidersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}
	sciInvalidJSON, node := mockStateContextI(), mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sciInvalidJSON.InsertTrieNode(AllProvidersKey, &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		msc   *MagmaSmartContract
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		want  interface{}
		error error
	}{
		{
			name:  "OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sci,
			want:  list.Nodes.Sorted,
			error: nil,
		},
		{
			name:  "Not_Present_OK",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   mockStateContextI(),
			want:  Providers{Nodes: &providersSorted{}}.Nodes.Sorted,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			msc:   msc,
			ctx:   nil,
			vals:  nil,
			sci:   sciInvalidJSON,
			want:  nil,
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.allProviders(test.ctx, test.vals, test.sci)
			if !errIs(err, test.error) {
				t.Errorf("allProviders() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("allProviders() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_billing(t *testing.T) {
	t.Parallel()

	msc, bill, sci := mockMagmaSmartContract(), mockBilling(), mockStateContextI()
	if _, err := sci.InsertTrieNode(bill.uid(msc.ID), bill); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	billInvalid := Billing{SessionID: "invalid_json_id"}
	nodeInvalidJSON := mockInvalidJson{ID: billInvalid.SessionID}
	if _, err := sci.InsertTrieNode(billInvalid.uid(msc.ID), &nodeInvalidJSON); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  *Billing
		error error
	}{
		{
			name:  "OK",
			id:    bill.SessionID,
			sci:   sci,
			msc:   msc,
			want:  bill,
			error: nil,
		},
		{
			name:  "Node_Not_Found_ERR",
			id:    "",
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: util.ErrNodeNotFound,
		},
		{
			name:  "Decode_ERR",
			id:    nodeInvalidJSON.ID,
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.billing(test.id, test.sci)
			if !errIs(err, test.error) {
				t.Errorf("billing() error: %v | want: %v", err, test.error)
				return
			}
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("billing() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_billingFetch(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	ackn := mockAcknowledgment()
	if _, err := sci.InsertTrieNode(ackn.uid(msc.ID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	bill := mockBilling()
	if _, err := sci.InsertTrieNode(bill.uid(msc.ID), bill); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		ctx   context.Context
		vals  url.Values
		sci   chain.StateContextI
		msc   *MagmaSmartContract
		want  interface{}
		error error
	}{
		{
			name:  "OK",
			ctx:   nil,
			vals:  url.Values{"id": {ackn.SessionID}},
			sci:   sci,
			msc:   msc,
			want:  bill,
			error: nil,
		},
		{
			name:  "Node_Not_Found_ERR",
			ctx:   nil,
			vals:  url.Values{},
			sci:   sci,
			msc:   msc,
			want:  nil,
			error: util.ErrNodeNotFound,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := test.msc.billingFetch(test.ctx, test.vals, test.sci)
			if !errIs(err, test.error) {
				t.Errorf("billingFetch() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("billingFetch() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_MagmaSmartContract_consumerAcceptTerms(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	cons, consList := mockConsumer(), Consumers{Nodes: &consumersSorted{}}
	if err := consList.add(msc.ID, cons, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	prov, provList := mockProvider(), Providers{Nodes: &providersSorted{}}
	if err := provList.add(msc.ID, prov, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	ackn := Acknowledgment{
		SessionID:     "session_id",
		AccessPointID: "access_point_id",
		Consumer:      cons,
		Provider:      prov,
	}

	blob := ackn.Encode()
	ackn.Consumer.ID = cons.ID

	prov.Terms.GetVolume()
	ackn.Provider.Terms = prov.Terms

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
			txn:   &tx.Transaction{ClientID: cons.ID, ToClientID: msc.ID},
			blob:  blob,
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

			got, err := test.msc.consumerAcceptTerms(test.txn, test.blob, test.sci)
			if (err != nil) != test.error {
				t.Errorf("consumerAcceptTerms() error: %v | want: %v", err, test.error)
				return
			}
			if got != test.want {
				t.Errorf("consumerAcceptTerms() got: %v | want: %v", got, test.want)
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
			sci:   sci,
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "Consumer_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_id"},
			blob:  nil,
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

func Test_MagmaSmartContract_consumerSessionStop(t *testing.T) {
	t.Parallel()

	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	cons, consList := mockConsumer(), Consumers{Nodes: &consumersSorted{}}
	if err := consList.add(msc.ID, cons, sci); err != nil {
		t.Fatalf("Consumers.add() error: %v | want: %v", err, nil)
	}

	prov, provList := mockProvider(), Providers{Nodes: &providersSorted{}}
	if err := provList.add(msc.ID, prov, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	ackn, bill := mockAcknowledgment(), mockBilling()
	bill.CalcAmount(ackn.Provider.Terms)
	if _, err := sci.InsertTrieNode(bill.uid(msc.ID), bill); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}
	bill.CompletedAt = common.Now()

	pool := tokenPool{
		PayerID: ackn.Consumer.ID,
		PayeeID: ackn.Provider.ID,
	}
	pool.ID = ackn.SessionID
	pool.Balance = 1000
	if _, err := sci.InsertTrieNode(pool.uid(msc.ID), &pool); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
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
			txn:   &tx.Transaction{ClientID: cons.ID, ToClientID: msc.ID},
			blob:  ackn.Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(bill.Encode()),
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

	cons, list := mockConsumer(), Consumers{Nodes: &consumersSorted{}}
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
	if _, err := sci.InsertTrieNode(ackn.uid(msc.ID), ackn); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	prov, provList := mockProvider(), Providers{Nodes: &providersSorted{}}
	if err := provList.add(msc.ID, prov, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	bill := mockBilling()
	bill.CalcAmount(ackn.Provider.Terms)

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
			blob:  bill.DataUsage.Encode(),
			sci:   sci,
			msc:   msc,
			want:  string(bill.Encode()),
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
			blob:  blob,
			sci:   sci,
			msc:   msc,
			want:  "",
			error: true,
		},
		{
			name:  "Provider_Insert_Trie_Node_ERR",
			txn:   &tx.Transaction{ClientID: "cannot_insert_id"},
			blob:  blob,
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

	prov, provList := mockProvider(), Providers{Nodes: &providersSorted{}}
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

	prov, list := mockProvider(), Providers{Nodes: &providersSorted{}}
	if err := list.add(msc.ID, prov, sci); err != nil {
		t.Fatalf("Providers.add() error: %v | want: %v", err, nil)
	}

	prov = mockProvider()
	prov.Terms.increase()
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
		ackn  *Acknowledgment
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
			ackn:  &Acknowledgment{SessionID: "not_present_id"},
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
		if got := nodeUID(scID, nodeID, nodeType); got != wantUID {
			t.Errorf("nodeUID() got: %v | want: %v", got, wantUID)
		}
	})
}
