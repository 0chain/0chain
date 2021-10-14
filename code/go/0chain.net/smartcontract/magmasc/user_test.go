package magmasc

import (
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

func Test_userFetch(t *testing.T) {
	t.Parallel()

	msc, sci, user := mockMagmaSmartContract(), mockStateContextI(), mockUser()
	if _, err := sci.InsertTrieNode(nodeUID(Address, userType, user.Id), user); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	node := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sci.InsertTrieNode(nodeUID(Address, userType, node.ID), &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    string
		sci   chain.StateContextI
		want  *zmc.User
		error bool
	}{
		{
			name:  "OK",
			id:    user.Id,
			sci:   sci,
			want:  user,
			error: false,
		},
		{
			name:  "Not_Present_ERR",
			id:    "not_present_id",
			sci:   sci,
			want:  nil,
			error: true,
		},
		{
			name:  "Decode_ERR",
			id:    node.ID,
			sci:   sci,
			want:  nil,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := userFetch(Address, test.id, msc.db, test.sci)
			if err == nil && !reflect.DeepEqual(got.Encode(), test.want.Encode()) {
				t.Errorf("userFetch() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("userFetch() error: %v | want: %v", err, test.error)
			}
		})
	}
}
