package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

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
		name    string
		blob    []byte
		want    Providers
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

			got := Providers{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
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
		list Providers
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
	if _, err := sci.InsertTrieNode(AllProvidersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	prov := mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ID, providerType), &prov); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	provInvalid := mockProvider()
	provInvalid.ID = "provider_invalid_id"
	provInvalid.Terms.ExpiredAt = 0
	if _, err := sci.InsertTrieNode(nodeUID(scID, provInvalid.ID, providerType), &provInvalid); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		prov  *Provider
		list  Providers
		sci   chain.StateContextI
		error bool
	}{
		{
			name:  "OK",
			prov:  &prov,
			list:  list,
			sci:   sci,
			error: false,
		},
		{
			name:  "Decode_ERR",
			prov:  &provInvalid,
			list:  list,
			sci:   sci,
			error: true,
		},
		{
			name:  "Internal_Unexpected_ERR",
			prov:  &Provider{ID: "unexpected_id"},
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

func Test_extractProviders(t *testing.T) {
	t.Parallel()

	sci, list := mockStateContextI(), mockProviders()
	if _, err := sci.InsertTrieNode(AllProvidersKey, &list); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		want  *Providers
		error error
	}{
		{
			name:  "OK",
			id:    AllProvidersKey,
			sci:   sci,
			want:  &list,
			error: nil,
		},
		{
			name:  "Not_Present_OK",
			id:    "not_present_id",
			sci:   mockStateContextI(),
			want:  &Providers{Nodes: &providersSorted{}},
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

			got, err := extractProviders(test.id, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("extractProviders() got: %#v | want: %#v", got, test.want)
				return
			}
			if !errIs(err, test.error) {
				t.Errorf("extractProviders() error: %v | want: %v", err, test.error)
			}
		})
	}
}
