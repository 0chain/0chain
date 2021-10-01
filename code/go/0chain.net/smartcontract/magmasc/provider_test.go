package magmasc

import (
	"reflect"
	"testing"

	zmc "github.com/0chain/gosdk/zmagmacore/magmasc"

	chain "0chain.net/chaincore/chain/state"
)

func Test_providerFetch(t *testing.T) {
	t.Parallel()

	msc, sci, prov := mockMagmaSmartContract(), mockStateContextI(), mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(Address, providerType, prov.ExtID), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	node := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sci.InsertTrieNode(nodeUID(Address, providerType, node.ID), &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    string
		sci   chain.StateContextI
		want  *zmc.Provider
		error bool
	}{
		{
			name:  "OK",
			id:    prov.ExtID,
			sci:   sci,
			want:  prov,
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

			got, err := providerFetch(Address, test.id, msc.db, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("providerFetch() got: %#v | want: %#v", err, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("providerFetch() error: %v | want: %v", err, test.error)
			}
		})
	}
}
