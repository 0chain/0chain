package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func Test_Consumers_Decode(t *testing.T) {
	t.Parallel()

	list := mockConsumers()
	blob, err := json.Marshal(list.Nodes.Sorted)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		blob    []byte
		want    Consumers
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: list,
		},
		{
			name:    "Decode_ERR",
			blob:    []byte(":"), // invalid json
			wantErr: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := Consumers{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_Encode(t *testing.T) {
	t.Parallel()

	list := mockConsumers()
	blob, err := json.Marshal(list.Nodes.Sorted)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		list Consumers
		want []byte
	}{
		{
			name: "OK",
			list: list,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumers_add(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	list, sci := mockConsumers(), mockStateContextI()
	if _, err := sci.InsertTrieNode(AllConsumersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	cons := mockConsumer()
	if _, err := sci.InsertTrieNode(nodeUID(scID, cons.ID, consumerType), cons); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		cons  *Consumer
		list  Consumers
		sci   chain.StateContextI
		error bool
	}{
		{
			name:  "OK",
			cons:  cons,
			list:  list,
			sci:   sci,
			error: false,
		},
		{
			name:  "Insert_Trie_Node_ERR",
			cons:  &Consumer{ExtID: "cannot_insert_id"},
			list:  list,
			sci:   sci,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.add(scID, test.cons, test.sci); (err != nil) != test.error {
				t.Errorf("add() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_fetchConsumers(t *testing.T) {
	t.Parallel()

	sci, list := mockStateContextI(), mockConsumers()
	if _, err := sci.InsertTrieNode(AllConsumersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		want  *Consumers
		error error
	}{
		{
			name:  "OK",
			id:    AllConsumersKey,
			sci:   sci,
			want:  &list,
			error: nil,
		},
		{
			name:  "Not_Present_OK",
			id:    "not_present_id",
			sci:   mockStateContextI(),
			want:  &Consumers{Nodes: &consumersSorted{}},
			error: nil,
		},
		{
			name:  "Decode_ERR",
			id:    "invalid_json_id",
			sci:   sci,
			want:  nil,
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := fetchConsumers(test.id, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("fetchConsumers() got: %#v | want: %#v", got, test.want)
				return
			}
			if !errIs(err, test.error) {
				t.Errorf("fetchConsumers() error: %v | want: %v", err, test.error)
			}
		})
	}
}
