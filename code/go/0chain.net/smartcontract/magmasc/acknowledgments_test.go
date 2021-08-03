package magmasc

import (
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	"0chain.net/core/datastore"
	store "0chain.net/core/ememorystore"
)

func TestActiveAcknowledgments_append(t *testing.T) {
	t.Parallel()

	const size = 10
	ackn, msc := mockAcknowledgment(), mockMagmaSmartContract()
	want := mockActiveAcknowledgments(size)
	want.Nodes = append(want.Nodes, ackn)

	tests := [1]struct {
		name  string
		list  *ActiveAcknowledgments
		ackn  *bmp.Acknowledgment
		conn  *store.Connection
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			list:  mockActiveAcknowledgments(size),
			ackn:  ackn,
			conn:  store.GetTransaction(msc.db),
			want:  want,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.append(test.ackn, test.conn); (err != nil) != test.error {
				t.Errorf("append() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func TestActiveAcknowledgments_remove(t *testing.T) {
	t.Parallel()

	const size = 10
	ackn, msc := mockAcknowledgment(), mockMagmaSmartContract()
	list := mockActiveAcknowledgments(size)
	list.Nodes = append(list.Nodes, ackn)

	tests := [1]struct {
		name  string
		list  *ActiveAcknowledgments
		ackn  *bmp.Acknowledgment
		conn  *store.Connection
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			list:  list,
			ackn:  ackn,
			conn:  store.GetTransaction(msc.db),
			want:  mockActiveAcknowledgments(size),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.remove(test.ackn, test.conn); (err != nil) != test.error {
				t.Errorf("remove() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_fetchActiveAcknowledgments(t *testing.T) {
	t.Parallel()

	const size = 10
	list, msc := mockActiveAcknowledgments(size), mockMagmaSmartContract()
	if err := list.append(mockAcknowledgment(), store.GetTransaction(msc.db)); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		id    datastore.Key
		conn  *store.Connection
		want  *ActiveAcknowledgments
		error bool
	}{
		{
			name:  "OK",
			id:    ActiveAcknowledgmentsKey,
			conn:  store.GetTransaction(msc.db),
			want:  list,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			// do not use parallel running to avoid detect race conditions because of
			// everything is happening in a single smart contract so there is only one thread
			got, err := fetchActiveAcknowledgments(test.id, test.conn)
			if (err != nil) != test.error {
				t.Errorf("fetchActiveAcknowledgments() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("fetchActiveAcknowledgments() got: %#v | want: %#v", got, test.want)
				t.Errorf("fetchActiveAcknowledgments() got len: %v | want len: %v", len(got.Nodes), len(test.want.Nodes))
			}
		})
	}
}
