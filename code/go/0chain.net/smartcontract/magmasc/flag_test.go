package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_flagBool_Decode(t *testing.T) {
	t.Parallel()

	flag := flagBool(true)
	blob, err := json.Marshal(flag)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [2]struct {
		name  string
		blob  []byte
		want  flagBool
		error bool
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  flag,
			error: false,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  flagBool(false),
			error: true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := flagBool(false)
			if err := got.Decode(test.blob); (err != nil) != test.error {
				t.Errorf("Decode() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_flagBool_Encode(t *testing.T) {
	t.Parallel()

	flag := flagBool(true)
	blob, err := json.Marshal(flag)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		flag flagBool
		want []byte
	}{
		{
			name: "OK",
			flag: flag,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.flag.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}
