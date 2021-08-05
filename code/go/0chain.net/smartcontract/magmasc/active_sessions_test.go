package magmasc

import (
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	store "0chain.net/core/ememorystore"
)

func Test_ActiveSessions_add(t *testing.T) {
	t.Parallel()

	const size = 10
	ackn, msc := mockAcknowledgment(), mockMagmaSmartContract()
	want := mockActiveSessions(size)
	want.Items = append(want.Items, ackn)

	tests := [1]struct {
		name  string
		msc   *MagmaSmartContract
		ackn  *bmp.Acknowledgment
		conn  *store.Connection
		sci   chain.StateContextI
		list  *ActiveSessions
		want  *ActiveSessions
		error bool
	}{
		{
			name:  "OK",
			msc:   msc,
			ackn:  ackn,
			conn:  store.GetTransaction(msc.db),
			sci:   mockStateContextI(),
			list:  mockActiveSessions(size),
			want:  want,
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			err := test.list.add(test.msc.ID, test.ackn, test.conn, test.sci)
			if (err != nil) != test.error {
				t.Errorf("append() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_ActiveSessions_remove(t *testing.T) {
	t.Parallel()

	const size = 10
	ackn, msc := mockAcknowledgment(), mockMagmaSmartContract()
	list := mockActiveSessions(size)
	list.Items = append(list.Items, ackn)

	tests := [1]struct {
		name  string
		ackn  *bmp.Acknowledgment
		conn  *store.Connection
		list  *ActiveSessions
		want  *ActiveSessions
		error bool
	}{
		{
			name:  "OK",
			list:  list,
			ackn:  ackn,
			conn:  store.GetTransaction(msc.db),
			want:  mockActiveSessions(size),
			error: false,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.del(test.ackn, test.conn); (err != nil) != test.error {
				t.Errorf("del() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_fetchActiveSessions(t *testing.T) {
	t.Parallel()

	const size = 10
	msc, sci := mockMagmaSmartContract(), mockStateContextI()

	t.Run("Not_Present_OK", func(t *testing.T) {
		// do not use parallel running to avoid detect race conditions because of
		// everything is happening in a single smart contract so there is only one thread
		got, err := fetchActiveSessions(ActiveSessionsKey, store.GetTransaction(msc.db))
		if err != nil {
			t.Errorf("fetchActiveSessions() error: %v | want: %v", err, nil)
			return
		}
		want := &ActiveSessions{}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("fetchActiveSessions() got: %#v | want: %#v", got, want)
		}
	})

	list := mockActiveSessions(size)
	if err := list.add(msc.ID, mockAcknowledgment(), store.GetTransaction(msc.db), sci); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		id    string
		conn  *store.Connection
		want  *ActiveSessions
		error bool
	}{
		{
			name:  "OK",
			id:    ActiveSessionsKey,
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
			got, err := fetchActiveSessions(test.id, test.conn)
			if (err != nil) != test.error {
				t.Errorf("fetchActiveSessions() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("fetchActiveSessions() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}
