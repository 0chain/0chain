package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_DataUsage_Decode(t *testing.T) {
	t.Parallel()

	dataUsage := mockDataUsage()
	blob, err := json.Marshal(dataUsage)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	dataUsageInvalid := mockDataUsage()
	dataUsageInvalid.SessionID = ""
	blobInvalid, err := json.Marshal(dataUsageInvalid)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		blob  []byte
		want  *DataUsage
		error error
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  dataUsage,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &DataUsage{},
			error: errDecodeData,
		},
		{
			name:  "Invalid_ERR",
			blob:  blobInvalid,
			want:  &DataUsage{},
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &DataUsage{}
			if err = got.Decode(test.blob); !errIs(err, test.error) {
				t.Errorf("Decode() error: %v | want: %v", err, test.error)
				return
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_Encode(t *testing.T) {
	t.Parallel()

	dataUsage := mockDataUsage()
	blob, err := json.Marshal(dataUsage)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name      string
		dataUsage *DataUsage
		want      []byte
	}{
		{
			name:      "OK",
			dataUsage: dataUsage,
			want:      blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.dataUsage.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_DataUsage_validate(t *testing.T) {
	t.Parallel()

	duEmptySessionID := mockDataUsage()
	duEmptySessionID.SessionID = ""

	tests := [2]struct {
		name      string
		dataUsage *DataUsage
		want      error
	}{
		{
			name:      "OK",
			dataUsage: mockDataUsage(),
			want:      nil,
		},
		{
			name:      "EmptySessionID",
			dataUsage: duEmptySessionID,
			want:      errDataUsageInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.dataUsage.validate(); !errIs(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}
