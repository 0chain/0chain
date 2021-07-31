package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func TestActiveAcknowledgments_Decode(t *testing.T) {
	t.Parallel()

	const size = 10
	list := mockActiveAcknowledgments(size)

	list.mutex.RLock()
	blob, err := json.Marshal(list)
	list.mutex.RUnlock()
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		blob  []byte
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  list,
			error: false,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &ActiveAcknowledgments{},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &ActiveAcknowledgments{}
			if err := got.Decode(test.blob); (err != nil) != test.error {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func TestActiveAcknowledgments_Encode(t *testing.T) {
	t.Parallel()

	const size = 10
	list := mockActiveAcknowledgments(size)

	list.mutex.RLock()
	blob, err := json.Marshal(list)
	list.mutex.RUnlock()
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		list *ActiveAcknowledgments
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
				t.Errorf("Encode() got: %v | want: %v", string(got), string(test.want))
			}
		})
	}
}

func TestActiveAcknowledgments_append(t *testing.T) {
	t.Parallel()

	const size = 10

	ackn, sci := mockAcknowledgment(), mockStateContextI()

	list := mockActiveAcknowledgments(size)
	if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	list.mutex.RLock()
	want := &ActiveAcknowledgments{Nodes: make([]*bmp.Acknowledgment, len(list.Nodes)+1)}
	copy(want.Nodes[:], list.Nodes[:])
	list.mutex.RUnlock()
	want.Nodes = append(want.Nodes, ackn)

	tests := [1]struct {
		name  string
		list  *ActiveAcknowledgments
		ackn  *bmp.Acknowledgment
		sci   state.StateContextI
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			list:  list,
			ackn:  ackn,
			sci:   sci,
			want:  want,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.append(test.ackn, test.sci); (err != nil) != test.error {
				t.Errorf("append() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func TestActiveAcknowledgments_remove(t *testing.T) {
	t.Parallel()

	const size = 10

	list, sci := mockActiveAcknowledgments(size), mockStateContextI()
	if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	list.mutex.RLock()
	ackn := *list.Nodes[0]
	want := &ActiveAcknowledgments{Nodes: make([]*bmp.Acknowledgment, len(list.Nodes)-1)}
	copy(want.Nodes[:], list.Nodes[1:]) // get rid item with index: 0
	list.mutex.RUnlock()

	tests := [1]struct {
		name  string
		list  *ActiveAcknowledgments
		ackn  *bmp.Acknowledgment
		sci   state.StateContextI
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			list:  list,
			ackn:  &ackn,
			sci:   sci,
			want:  want,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.remove(test.ackn, test.sci); (err != nil) != test.error {
				t.Errorf("remove() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_fetchActiveAcknowledgments(t *testing.T) {
	t.Parallel()

	const size = 10

	list, sci := mockActiveAcknowledgments(size), mockStateContextI()
	if _, err := sci.InsertTrieNode(ActiveAcknowledgmentsKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		id    datastore.Key
		sci   state.StateContextI
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			id:    ActiveAcknowledgmentsKey,
			sci:   sci,
			want:  list,
			error: false,
		},
		{
			name:  "Decode_ERR",
			id:    "invalid_json_id",
			sci:   sci,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := fetchActiveAcknowledgments(test.id, test.sci)
			if (err != nil) != test.error {
				t.Errorf("fetchActiveAcknowledgments() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("fetchActiveAcknowledgments() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}
