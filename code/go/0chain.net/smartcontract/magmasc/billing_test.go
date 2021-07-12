package magmasc

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
)

func Test_Billing_CalcAmount(t *testing.T) {
	t.Parallel()

	bill, cost := mockBilling(), uint64(1*billion)

	bps := big.NewFloat(0).Add( // data usage summary: UploadBytes + DownloadBytes
		big.NewFloat(0).SetUint64(bill.DataUsage.UploadBytes),
		big.NewFloat(0).SetUint64(bill.DataUsage.DownloadBytes),
	)

	want, _ := big.NewFloat(0).Mul(
		big.NewFloat(0).Quo(bps, big.NewFloat(million)), // data usage in mega bytes
		big.NewFloat(0).SetUint64(cost),                 // cost per mega byte
	).Int64() // rounded of amount for data usage multiplied by cost

	tests := [2]struct {
		name string
		bill *Billing
		cost uint64
		want int64
	}{
		{
			name: "OK",
			bill: bill,
			cost: cost,
			want: want,
		},
		{
			name: "Zero_OK",
			bill: mockBilling(),
			cost: 0,
			want: 0,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.bill.Amount != 0 { // must be zero before first call CalcAmount()
				t.Errorf("Billing.Amount is: %v | want: %v", test.bill.Amount, 0)
			}

			test.bill.CalcAmount(test.cost)
			if test.bill.Amount != test.want { // must be the same value with test.want after called CalcAmount()
				t.Errorf("GetVolume() got: %v | want: %v", test.bill.Amount, test.want)
			}
		})
	}
}

func Test_Billing_Decode(t *testing.T) {
	t.Parallel()

	bill := mockBilling()
	blob, err := json.Marshal(bill)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	billInvalid := mockBilling()
	billInvalid.DataUsage.SessionID = ""
	blobInvalid, err := json.Marshal(billInvalid)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [3]struct {
		name  string
		blob  []byte
		want  *Billing
		error error
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  bill,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  &Billing{},
			error: errDecodeData,
		},
		{
			name:  "Invalid_ERR",
			blob:  blobInvalid,
			want:  &Billing{},
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := &Billing{}
			if err = got.Decode(test.blob); !errIs(err, test.error) {
				t.Errorf("Decode() error: %v | want: %v", err, nil)
				return
			}
			if test.want != nil && !reflect.DeepEqual(got, test.want) {
				t.Errorf("Decode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Billing_Encode(t *testing.T) {
	t.Parallel()

	bill := mockBilling()
	blob, err := json.Marshal(bill)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name string
		bill *Billing
		want []byte
	}{
		{
			name: "OK",
			bill: bill,
			want: blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.bill.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_Billing_uid(t *testing.T) {
	t.Parallel()

	const (
		scID      = "sc_uid"
		sessionID = "session_id"
		billUID   = "sc:" + scID + ":datausage:" + sessionID
	)

	t.Run("OK", func(t *testing.T) {
		t.Parallel()

		bill := Billing{SessionID: sessionID}
		if got := bill.uid(scID); got != billUID {
			t.Errorf("uid() got: %v | want: %v", got, billUID)
		}
	})
}

func Test_Billing_validate(t *testing.T) {
	t.Parallel()

	bill, dataUsage := mockBilling(), mockDataUsage()

	billNilDataUsage := mockBilling()
	billNilDataUsage.DataUsage = nil

	duInvalidSessionID := mockDataUsage()
	duInvalidSessionID.SessionID = "not_present_id"

	duInvalidSessionTime := mockDataUsage()
	duInvalidSessionTime.SessionTime = bill.DataUsage.SessionTime - 1

	duInvalidUploadBytes := mockDataUsage()
	duInvalidUploadBytes.UploadBytes = bill.DataUsage.UploadBytes - 1

	duInvalidDownloadBytes := mockDataUsage()
	duInvalidDownloadBytes.DownloadBytes = bill.DataUsage.DownloadBytes - 1

	tests := [7]struct {
		name string
		du   *DataUsage
		bill *Billing
		want error
	}{
		{
			name: "OK",
			du:   dataUsage,
			bill: bill,
			want: nil,
		},
		{
			name: "nil_Billing_Data_Usage_OK",
			du:   dataUsage,
			bill: billNilDataUsage,
			want: nil,
		},
		{
			name: "nil_Data_Usage_ERR",
			du:   nil,
			bill: bill,
			want: errDataUsageInvalid,
		},
		{
			name: "Invalid_Session_ID_ERR",
			du:   duInvalidSessionID,
			bill: bill,
			want: errDataUsageInvalid,
		},
		{
			name: "Invalid_Session_Time_ERR",
			du:   duInvalidSessionTime,
			bill: bill,
			want: errDataUsageInvalid,
		},
		{
			name: "Invalid_Upload_Bytes_ERR",
			du:   duInvalidUploadBytes,
			bill: bill,
			want: errDataUsageInvalid,
		},
		{
			name: "Invalid_Download_Bytes_ERR",
			du:   duInvalidDownloadBytes,
			bill: bill,
			want: errDataUsageInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.bill.validate(test.du); !errIs(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}
