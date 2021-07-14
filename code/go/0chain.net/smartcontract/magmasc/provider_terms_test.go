package magmasc

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"

	"0chain.net/chaincore/state"
	"0chain.net/core/common"
)

func Test_ProviderTerms_Decode(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	blob, err := json.Marshal(terms)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	termsInvalid := mockProviderTerms()
	termsInvalid.QoS.UploadMbps = -0.1
	uBlobInvalid, err := json.Marshal(termsInvalid)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	termsInvalid = mockProviderTerms()
	termsInvalid.QoS.DownloadMbps = -0.1
	dBlobInvalid, err := json.Marshal(termsInvalid)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [4]struct {
		name  string
		blob  []byte
		want  ProviderTerms
		error error
	}{
		{
			name:  "OK",
			blob:  blob,
			want:  terms,
			error: nil,
		},
		{
			name:  "Decode_ERR",
			blob:  []byte(":"), // invalid json
			want:  ProviderTerms{},
			error: errDecodeData,
		},
		{
			name:  "QoS_Upload_Mbps_Invalid_ERR",
			blob:  uBlobInvalid,
			want:  ProviderTerms{},
			error: errDecodeData,
		},
		{
			name:  "QoS_Download_Mbps_Invalid_ERR",
			blob:  dBlobInvalid,
			want:  ProviderTerms{},
			error: errDecodeData,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ProviderTerms{}
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

func Test_ProviderTerms_Encode(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	blob, err := json.Marshal(terms)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v | want: %v", err, nil)
	}

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  []byte
	}{
		{
			name:  "OK",
			terms: terms,
			want:  blob,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.Encode(); !reflect.DeepEqual(got, test.want) {
				t.Errorf("Encode() got: %#v | want: %#v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_GetAmount(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()

	termsZeroPrice := mockProviderTerms()
	termsZeroPrice.Price = 0

	tests := [2]struct {
		name  string
		terms ProviderTerms
		want  state.Balance
	}{
		{
			name:  "OK",
			terms: terms,
			want:  state.Balance(terms.GetPrice() * terms.GetVolume()),
		},
		{
			name:  "Zero_OK",
			terms: termsZeroPrice,
			want:  0,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.GetAmount(); got != test.want {
				t.Errorf("GetAmount() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_GetMinCost(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	minCost, _ := big.NewFloat(0).Mul( // convert to token units price
		big.NewFloat(billion),
		big.NewFloat(float64(terms.MinCost)),
	).Int64() // rounded value of price multiplied by volume

	termsZeroMinCost := mockProviderTerms()
	termsZeroMinCost.MinCost = 0

	tests := [2]struct {
		name  string
		terms ProviderTerms
		want  int64
	}{
		{
			name:  "OK",
			terms: terms,
			want:  minCost,
		},
		{
			name:  "Zero_Min_Cost_OK",
			terms: termsZeroMinCost,
			want:  0,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.GetMinCost(); got != test.want {
				t.Errorf("GetMinCost() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_GetPrice(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	price, _ := big.NewFloat(0).Mul( // convert to token units price
		big.NewFloat(billion),
		big.NewFloat(float64(terms.Price)),
	).Int64() // rounded value of price multiplied by volume

	termsZeroPrice := mockProviderTerms()
	termsZeroPrice.Price = 0

	tests := [2]struct {
		name  string
		terms ProviderTerms
		want  int64
	}{
		{
			name:  "OK",
			terms: terms,
			want:  price,
		},
		{
			name:  "Zero_Price_OK",
			terms: termsZeroPrice,
			want:  0,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.GetPrice(); got != test.want {
				t.Errorf("GetPrice() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_GetVolume(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()
	mbps := big.NewFloat(0).Add( // provider terms summary: UploadMbps + DownloadMbps
		big.NewFloat(float64(terms.QoS.UploadMbps)),
		big.NewFloat(float64(terms.QoS.DownloadMbps)),
	)
	volume, _ := big.NewFloat(0).Mul(
		big.NewFloat(0).Quo(mbps, big.NewFloat(octet)),                // mega bytes per second
		big.NewFloat(0).SetInt64(int64(terms.ExpiredAt-common.Now())), // duration in seconds
	).Int64() // rounded of bytes per second multiplied by duration

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  int64
	}{
		{
			name:  "OK",
			terms: terms,
			want:  volume,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if test.terms.Volume != 0 { // must be zero before first call GetVolume()
				t.Errorf("ProviderTerms.Volume is: %v | want: %v", test.terms.Volume, 0)
			}
			if got := test.terms.GetVolume(); got != test.want {
				t.Errorf("GetVolume() got: %v | want: %v", got, test.want)
			}
			if test.terms.Volume != test.want { // must be the same value with test.want after called GetVolume()
				t.Errorf("ProviderTerms.Volume is: %v | want: %v", test.terms.Volume, test.want)
			}
		})
	}
}

func Test_ProviderTerms_decrease(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()

	if terms.AutoUpdateQoS.UploadMbps != 0 { // up the upload mbps quality of service
		upload := big.NewFloat(float64(terms.QoS.UploadMbps))
		update := big.NewFloat(float64(terms.AutoUpdateQoS.UploadMbps))
		terms.QoS.UploadMbps, _ = big.NewFloat(0).Add(upload, update).Float32()
	}
	if terms.AutoUpdateQoS.DownloadMbps != 0 { // up the download mbps quality of service
		download := big.NewFloat(float64(terms.QoS.DownloadMbps))
		update := big.NewFloat(float64(terms.AutoUpdateQoS.DownloadMbps))
		terms.QoS.DownloadMbps, _ = big.NewFloat(0).Add(download, update).Float32()
	}
	if terms.AutoUpdatePrice != 0 { // prepare price and auto update value
		price := big.NewFloat(float64(terms.Price))
		update := big.NewFloat(float64(terms.AutoUpdatePrice))
		if price.Cmp(update) == 1 { // check if the price is greater than the value of auto update
			terms.Price, _ = big.NewFloat(0).Sub(price, update).Float32() // down the price
		}
	}
	// prolong terms. expire
	terms.ExpiredAt += common.Timestamp(terms.ProlongDuration)
	// the volume of terms must to be zeroed
	terms.Volume = 0

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  ProviderTerms
	}{
		{
			name:  "OK",
			terms: mockProviderTerms(),
			want:  terms,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.terms.decrease()
			if !reflect.DeepEqual(test.terms, test.want) {
				t.Errorf("decrease() got: %#v | want: %#v", test.terms, test.want)
			}
		})
	}
}

func Test_ProviderTerms_expired(t *testing.T) {
	t.Parallel()

	termsValid := mockProviderTerms()

	termsExpired := mockProviderTerms()
	termsExpired.ExpiredAt = common.Now()

	tests := [2]struct {
		name  string
		terms ProviderTerms
		want  bool
	}{
		{
			name:  "FALSE",
			terms: termsValid,
			want:  false,
		},
		{
			name:  "TRUE",
			terms: termsExpired,
			want:  true,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if got := test.terms.expired(); got != test.want {
				t.Errorf("expired() got: %v | want: %v", got, test.want)
			}
		})
	}
}

func Test_ProviderTerms_increase(t *testing.T) {
	t.Parallel()

	terms := mockProviderTerms()

	if terms.AutoUpdatePrice != 0 { // up the price of service
		price := big.NewFloat(float64(terms.Price))
		update := big.NewFloat(float64(terms.AutoUpdatePrice))
		terms.Price, _ = big.NewFloat(0).Add(price, update).Float32()
	}
	if terms.AutoUpdateQoS.UploadMbps != 0 { // prepare upload mbps quality of service
		upload := big.NewFloat(float64(terms.QoS.UploadMbps))
		update := big.NewFloat(float64(terms.AutoUpdateQoS.UploadMbps))
		if upload.Cmp(update) == 1 { // down thr upload mbps quality of service
			terms.QoS.UploadMbps, _ = big.NewFloat(0).Sub(upload, update).Float32()
		}
	}
	if terms.AutoUpdateQoS.DownloadMbps != 0 { // prepare download mbps quality of service
		download := big.NewFloat(float64(terms.QoS.DownloadMbps))
		update := big.NewFloat(float64(terms.AutoUpdateQoS.DownloadMbps))
		if download.Cmp(update) == 1 { // down the download mbps quality of service
			terms.QoS.DownloadMbps, _ = big.NewFloat(0).Sub(download, update).Float32()
		}
	}
	// prolong expire of terms
	terms.ExpiredAt += common.Timestamp(terms.ProlongDuration)
	// the volume must to be zeroed
	terms.Volume = 0

	tests := [1]struct {
		name  string
		terms ProviderTerms
		want  ProviderTerms
	}{
		{
			name:  "OK",
			terms: mockProviderTerms(),
			want:  terms,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			test.terms.increase()
			if !reflect.DeepEqual(test.terms, test.want) {
				t.Errorf("increase() got: %#v | want: %#v", test.terms, test.want)
			}
		})
	}
}

func Test_ProviderTerms_validate(t *testing.T) {
	t.Parallel()

	termsZeroExpiredAt := mockProviderTerms()
	termsZeroExpiredAt.ExpiredAt = 0

	termsZeroQoSUploadMbps := mockProviderTerms()
	termsZeroQoSUploadMbps.QoS.UploadMbps = 0

	termsZeroQoSDownloadMbps := mockProviderTerms()
	termsZeroQoSDownloadMbps.QoS.DownloadMbps = 0

	tests := [4]struct {
		name  string
		terms ProviderTerms
		want  error
	}{
		{
			name:  "OK",
			terms: mockProviderTerms(),
			want:  nil,
		},
		{
			name:  "Zero_Expired_At_ERR",
			terms: termsZeroExpiredAt,
			want:  errProviderTermsInvalid,
		},
		{
			name:  "Zero_QoS_Upload_Mbps_ERR",
			terms: termsZeroQoSUploadMbps,
			want:  errProviderTermsInvalid,
		},
		{
			name:  "Zero_QoS_Download_Mbps_ERR",
			terms: termsZeroQoSDownloadMbps,
			want:  errProviderTermsInvalid,
		},
	}

	for idx := range tests {
		test := tests[idx]
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if err := test.terms.validate(); !errIs(err, test.want) {
				t.Errorf("validate() error: %v | want: %v", err, test.want)
			}
		})
	}
}
