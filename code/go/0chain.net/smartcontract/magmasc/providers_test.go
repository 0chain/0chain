package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	bmp "github.com/0chain/bandwidth_marketplace/code/core/magmasc"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
)

func Test_Providers_Decode(t *testing.T) {
	t.Parallel()

	list := mockProviders()
	blob, err := json.Marshal(list.Nodes.Sorted)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		blob  []byte
		want  *Providers
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
			want:  &Providers{},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &Providers{}
			if err := got.Decode(test.blob); (err != nil) != test.error {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Providers_Encode(t *testing.T) {
	t.Parallel()

	list := mockProviders()
	blob, err := json.Marshal(list.Nodes.Sorted)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		list *Providers
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

func Test_Providers_add(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"
	list, sci := mockProviders(), mockStateContextI()
	if _, err := sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	prov := mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ExtID, providerType), prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	provRegistered, _ := list.Nodes.getByIndex(0)

	tests := [4]struct {
		name  string
		prov  *bmp.Provider
		list  *Providers
		sci   chain.StateContextI
		error bool
	}{
		{
			name:  "OK",
			prov:  prov,
			list:  list,
			sci:   sci,
			error: false,
		},
		{
			name:  "Provider_Host_Already_Registered_ERR",
			prov:  provRegistered,
			list:  list,
			sci:   sci,
			error: true,
		},
		{
			name:  "Provider_Insert_Trie_Node_ERR",
			prov:  &bmp.Provider{ExtID: "cannot_insert_id"},
			list:  list,
			sci:   sci,
			error: true,
		},
		{
			name:  "List_Insert_Trie_Node_ERR",
			prov:  &bmp.Provider{ExtID: "cannot_insert_list"},
			list:  list,
			sci:   sci,
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.list.add(scID, test.prov, test.sci); (err != nil) != test.error {
				t.Errorf("add() error: %v | want: %v", err, test.error)
			}
		})
	}
}

func Test_fetchProviders(t *testing.T) {
	t.Parallel()

	sci, list := mockStateContextI(), mockProviders()
	if _, err := sci.InsertTrieNode(AllProvidersKey, list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		want  *Providers
		error bool
	}{
		{
			name:  "OK",
			id:    AllProvidersKey,
			sci:   sci,
			want:  list,
			error: false,
		},
		{
			name:  "Not_Present_OK",
			id:    "not_present_id",
			sci:   mockStateContextI(),
			want:  &Providers{},
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

			got, err := fetchProviders(test.id, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("fetchProviders() got: %#v | want: %#v", got, test.want)
				return
			}
			if (err != nil) != test.error {
				t.Errorf("fetchProviders() error: %v | want: %v", err, test.error)
			}
		})
	}
}
