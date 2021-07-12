package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func Test_Provider_Decode(t *testing.T) {
	t.Parallel()

	prov := mockProvider()
	blob, err := json.Marshal(prov)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name    string
		blob    []byte
		want    Provider
		wantErr bool
	}{
		{
			name: "OK",
			blob: blob,
			want: prov,
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

			got := Provider{}
			if err = got.Decode(test.blob); (err != nil) != test.wantErr {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Provider_Encode(t *testing.T) {
	t.Parallel()

	prov := mockProvider()
	blob, err := json.Marshal(prov)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		prov Provider
		want []byte
	}{
		{
			name: "OK",
			prov: prov,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.prov.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Provider_GetType(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		prov := Provider{}
		if got := prov.GetType(); got != providerType {
			t.Errorf("GetType() got: %v | want: %v", got, providerType)
		}
	})
}

func Test_extractProvider(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	sci, prov := mockStateContextI(), mockProvider()
	if _, err := sci.InsertTrieNode(nodeUID(scID, prov.ID, providerType), &prov); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}
	node := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sci.InsertTrieNode(nodeUID(scID, node.ID, providerType), &node); err != nil {
		t.Fatalf("InsertTrieNode() got: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		want  *Provider
		error error
	}{
		{
			name:  "OK",
			id:    prov.ID,
			sci:   sci,
			want:  &prov,
			error: nil,
		},
		{
			name:  "Not_Present_ERR",
			id:    "not_present_id",
			sci:   sci,
			want:  nil,
			error: util.ErrValueNotPresent,
		},
		{
			name:  "Decode_ERR",
			id:    node.ID,
			sci:   sci,
			want:  nil,
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got, err := extractProvider(scID, test.id, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("extractProvider() got: %#v | want: %#v", err, test.want)
				return
			}
			if !errIs(test.error, err) {
				t.Errorf("extractProvider() error: %v | want: %v", err, test.error)
			}
		})
	}
}
