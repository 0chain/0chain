package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"

	chain "0chain.net/chaincore/chain/state"
	"0chain.net/core/datastore"
	"0chain.net/core/util"
)

func Test_Consumer_Decode(t *testing.T) {
	t.Parallel()

	cons := mockConsumer()
	blob, err := json.Marshal(cons)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		blob  []byte
		want  *Consumer
		error bool
	}{
		{
			name: "OK",
			blob: blob,
			want: cons,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &Consumer{},
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &Consumer{}
			if err = got.Decode(test.blob); (err != nil) != test.error {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumer_Encode(t *testing.T) {
	t.Parallel()

	cons := mockConsumer()
	blob, err := json.Marshal(cons)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		cons *Consumer
		want []byte
	}{
		{
			name: "OK",
			cons: cons,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.cons.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Consumer_GetType(t *testing.T) {
	t.Parallel()

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		cons := Consumer{}
		if got := cons.GetType(); got != consumerType {
			t.Errorf("GetType() got: %v | want: %v", got, consumerType)
		}
	})
}

func Test_consumerFetch(t *testing.T) {
	t.Parallel()

	const scID = "sc_id"

	sci, cons := mockStateContextI(), mockConsumer()
	if _, err := sci.InsertTrieNode(nodeUID(scID, cons.ExtID, consumerType), cons); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	node := mockInvalidJson{ID: "invalid_json_id"}
	if _, err := sci.InsertTrieNode(nodeUID(scID, node.ID, consumerType), &node); err != nil {
		t.Fatalf("InsertTrieNode() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		id    datastore.Key
		sci   chain.StateContextI
		want  *Consumer
		error error
	}{
		{
			name:  "OK",
			id:    cons.ExtID,
			sci:   sci,
			want:  cons,
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

			got, err := consumerFetch(scID, test.id, test.sci)
			if err == nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("consumerFetch() got: %#v | want: %#v", err, test.want)
				return
			}
			if !errIs(test.error, err) {
				t.Errorf("consumerFetch() error: %v | want: %v", err, test.error)
			}
		})
	}
}
