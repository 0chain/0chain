package encryption

import (
	"encoding/hex"
	"reflect"
	"testing"
)

func TestIsValidSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsValidSignatureScheme_ed25519_TRUE",
			args: args{sigScheme: "ed25519"},
			want: true,
		},
		{
			name: "Test_IsValidSignatureScheme_bls0chain_TRUE",
			args: args{sigScheme: "bls0chain"},
			want: true,
		},
		{
			name: "Test_IsValidSignatureScheme_FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsValidSignatureScheme(tt.args.sigScheme); got != tt.want {
				t.Errorf("IsValidSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
	}
	tests := []struct {
		name      string
		args      args
		want      SignatureScheme
		wantPanic bool
	}{
		{
			name: "Test_GetSignatureScheme_ed25519_OK",
			args: args{sigScheme: "ed25519"},
			want: NewED25519Scheme(),
		},
		{
			name: "Test_GetSignatureScheme_bls0chain_OK",
			args: args{sigScheme: "bls0chain"},
			want: NewBLS0ChainScheme(),
		},
		{
			name:      "Test_GetSignatureScheme_PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetSignatureScheme() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := GetSignatureScheme(tt.args.sigScheme); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidAggregateSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsValidAggregateSignatureScheme_ed25519_FAlSE",
			args: args{sigScheme: "ed25519"},
			want: false,
		},
		{
			name: "Test_IsValidAggregateSignatureScheme_bls0chain_TRUE",
			args: args{sigScheme: "bls0chain"},
			want: true,
		},
		{
			name: "Test_IsValidAggregateSignatureScheme_FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsValidAggregateSignatureScheme(tt.args.sigScheme); got != tt.want {
				t.Errorf("IsValidAggregateSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetAggregateSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
		total     int
		batchSize int
	}
	tests := []struct {
		name      string
		args      args
		want      AggregateSignatureScheme
		wantPanic bool
	}{
		{
			name: "Test_IsValidAggregateSignatureScheme_ed25519_OK",
			args: args{sigScheme: "ed25519"},
			want: nil,
		},
		{
			name: "Test_IsValidAggregateSignatureScheme_bls0chain_OK",
			args: args{sigScheme: "bls0chain", total: 2, batchSize: 1},
			want: NewBLS0ChainAggregateSignature(2, 1),
		},
		{
			name:      "Test_IsValidAggregateSignatureScheme_PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetAggregateSignatureScheme() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := GetAggregateSignatureScheme(tt.args.sigScheme, tt.args.total, tt.args.batchSize); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetAggregateSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsValidThresholdSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsValidThresholdSignatureScheme_ed25519_TRUE",
			args: args{sigScheme: "ed25519"},
			want: false,
		},
		{
			name: "Test_IsValidThresholdSignatureScheme_bls0chain_TRUE",
			args: args{sigScheme: "bls0chain"},
			want: true,
		},
		{
			name: "Test_IsValidThresholdSignatureScheme_FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsValidThresholdSignatureScheme(tt.args.sigScheme); got != tt.want {
				t.Errorf("IsValidThresholdSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetThresholdSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
	}
	tests := []struct {
		name      string
		args      args
		want      ThresholdSignatureScheme
		wantPanic bool
	}{
		{
			name: "Test_GetThresholdSignatureScheme_ed25519_OK",
			args: args{sigScheme: "ed25519"},
			want: nil,
		},
		{
			name: "Test_GetThresholdSignatureScheme_bls0chain_OK",
			args: args{sigScheme: "bls0chain"},
			want: NewBLS0ChainThresholdScheme(),
		},
		{
			name:      "Test_GetThresholdSignatureScheme_PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetThresholdSignatureScheme() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := GetThresholdSignatureScheme(tt.args.sigScheme); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetThresholdSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGenerateThresholdKeyShares(t *testing.T) {
	t.Parallel()

	s := NewBLS0ChainScheme()
	if err := s.GenerateKeys(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		sigScheme   string
		t           int
		n           int
		originalKey SignatureScheme
	}
	tests := []struct {
		name      string
		args      args
		want      []ThresholdSignatureScheme
		wantErr   bool
		wantPanic bool
	}{
		{
			name: "Test_GenerateThresholdKeyShares_ed25519_OK",
			args: args{sigScheme: "ed25519"},
			want: nil,
		},
		{
			name: "Test_GenerateThresholdKeyShares_bls0chain_OK",
			args: args{sigScheme: "bls0chain", originalKey: s, t: 2, n: 2},
		},
		{
			name:      "Test_GenerateThresholdKeyShares_PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetThresholdSignatureScheme() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			_, err := GenerateThresholdKeyShares(tt.args.sigScheme, tt.args.t, tt.args.n, tt.args.originalKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateThresholdKeyShares() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestIsValidReconstructSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsValidReconstructSignatureScheme_ed25519_TRUE",
			args: args{sigScheme: "ed25519"},
			want: false,
		},
		{
			name: "Test_IsValidReconstructSignatureScheme_bls0chain_TRUE",
			args: args{sigScheme: "bls0chain"},
			want: true,
		},
		{
			name: "Test_IsValidReconstructSignatureScheme_FALSE",
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsValidReconstructSignatureScheme(tt.args.sigScheme); got != tt.want {
				t.Errorf("IsValidReconstructSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetReconstructSignatureScheme(t *testing.T) {
	t.Parallel()

	type args struct {
		sigScheme string
		t         int
		n         int
	}
	tests := []struct {
		name      string
		args      args
		want      ReconstructSignatureScheme
		wantPanic bool
	}{
		{
			name: "Test_GetReconstructSignatureScheme_ed25519_OK",
			args: args{sigScheme: "ed25519"},
			want: nil,
		},
		{
			name: "Test_GetReconstructSignatureScheme_bls0chain_OK",
			args: args{sigScheme: "bls0chain"},
			want: NewBLS0ChainReconstruction(0, 0),
		},
		{
			name:      "Test_GetReconstructSignatureScheme_PANIC",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetReconstructSignatureScheme() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := GetReconstructSignatureScheme(tt.args.sigScheme, tt.args.t, tt.args.n); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetReconstructSignatureScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRawHash(t *testing.T) {
	t.Parallel()

	data := []byte("data")

	type args struct {
		hash interface{}
	}
	tests := []struct {
		name      string
		args      args
		want      []byte
		wantErr   bool
		wantPanic bool
	}{
		{
			name: "TestGetRawHash_Bytes_OK",
			args: args{hash: data},
			want: []byte("data"),
		},
		{
			name: "TestGetRawHash_String_OK",
			args: args{hash: hex.EncodeToString(data)},
			want: data,
		},
		{
			name:    "TestGetRawHash_String_ERR",
			args:    args{hash: "!"},
			wantErr: true,
		},
		{
			name:      "TestGetRawHash_Panic",
			args:      args{hash: 123},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetReconstructSignatureScheme() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got, err := GetRawHash(tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetRawHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRawHash() got = %v, want %v", got, tt.want)
			}
		})
	}
}
