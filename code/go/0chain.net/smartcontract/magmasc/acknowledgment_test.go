package magmasc

import (
	"encoding/json"
	"reflect"
	"testing"
)

func Test_Acknowledgment_Decode(t *testing.T) {
	t.Parallel()

	ackn := mockAcknowledgment()
	blob, err := json.Marshal(ackn)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	acknInvalid := mockAcknowledgment()
	acknInvalid.SessionID = ""
	blobInvalid, err := json.Marshal(acknInvalid)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		blob  []byte
		want  *Acknowledgment
		error error
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  ackn,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &Acknowledgment{},
			error: errDecodeData,
		},
		{
			name:  "Invalid_ERR",
			blob:  blobInvalid,
			want:  &Acknowledgment{},
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &Acknowledgment{}
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

func Test_Acknowledgment_Encode(t *testing.T) {
	t.Parallel()

	ackn := mockAcknowledgment()
	blob, err := json.Marshal(ackn)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		ackn *Acknowledgment
		want []byte
	}{
		{
			name: "OK",
			ackn: ackn,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.ackn.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Acknowledgment_uid(t *testing.T) {
	t.Parallel()

	const (
		scID      = "sc_uid"
		sessionID = "session_id"
		acknUID   = "sc:" + scID + ":acknowledgment:" + sessionID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		ackn := Acknowledgment{SessionID: sessionID}
		if got := ackn.uid(scID); got != acknUID {
			t.Errorf("uid() got: %v | want: %v", got, acknUID)
		}
	})
}

func Test_Acknowledgment_validate(t *testing.T) {
	t.Parallel()

	acknValid := mockAcknowledgment()

	acknEmptyAccessPointID := mockAcknowledgment()
	acknEmptyAccessPointID.AccessPointID = ""

	acknEmptyProviderID := mockAcknowledgment()
	acknEmptyProviderID.ProviderID = ""

	acknEmptySessionID := mockAcknowledgment()
	acknEmptySessionID.SessionID = ""

	tests := [4]struct {
		name string
		ackn *Acknowledgment
		want error
	}{
		{
			name: "OK",
			ackn: acknValid,
			want: nil,
		},
		{
			name: "Empty_Access_Point_ID",
			ackn: acknEmptyAccessPointID,
			want: errAcknowledgmentInvalid,
		},
		{
			name: "Empty_Provider_ID",
			ackn: acknEmptyProviderID,
			want: errAcknowledgmentInvalid,
		},
		{
			name: "Empty_Session_ID",
			ackn: acknEmptySessionID,
			want: errAcknowledgmentInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.ackn.validate(); !errIs(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}
