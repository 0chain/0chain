package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/chain/state"
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
			name:    "ERR",
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

func Test_Consumers_contains(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	list := mockConsumers()
	sci, cons := mockStateContextI(), mockConsumer()
	if _, err := sci.InsertTrieNode(nodeUID(scID, cons.ID, consumerType), &cons); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name string
		cons *Consumer
		list Consumers
		sci  state.StateContextI
		want bool
	}{
		{
			name: "FALSE",
			cons: &Consumer{ID: "not_present_id"},
			list: list,
			sci:  sci,
			want: false,
		},
		{
			name: "InNodeList_TRUE",
			cons: list.Nodes.Sorted[0],
			list: list,
			want: true,
		},
		{
			name: "InStateContext_TRUE",
			cons: &cons,
			list: list,
			sci:  sci,
			want: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.list.contains(scID, test.cons, test.sci); got != test.want {
				t.Errorf("contains() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_extractConsumers(t *testing.T) {
	t.Parallel()

	sci, list := mockStateContextI(), mockConsumers()
	if _, err := sci.InsertTrieNode(AllConsumersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   state.StateContextI
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

			got, err := extractConsumers(test.id, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("extractConsumers() got: %#v | want: %#v", got, test.want)
				return
			}
			if !errIs(err, test.error) {
				t.Errorf("extractConsumers() error: %v | want: %v", err, test.error)
			}
		})
	}
}
