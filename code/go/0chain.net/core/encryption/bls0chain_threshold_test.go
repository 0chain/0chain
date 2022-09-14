package encryption

import (
	"reflect"
	"testing"

	"github.com/herumi/bls/ffi/go/bls"

	"github.com/0chain/common/core/logging"
)

func init() {
	logging.InitLogging("development", "")
}

func TestThresholdSignatures(t *testing.T) {
	T := 7
	N := 10
	scheme := "bls0chain"
	msg := "1234567890"

	groupKey := GetSignatureScheme(scheme)
	err := groupKey.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	shares, err := GenerateThresholdKeyShares(scheme, T, N, groupKey)
	if err != nil {
		t.Fatal(err)
	}

	var sigs []string
	for _, share := range shares {
		sig, err := share.Sign(msg)
		if err != nil {
			t.Fatal(err)
		}

		sigs = append(sigs, sig)
	}

	rec := GetReconstructSignatureScheme(scheme, T, N)

	for i, share := range shares {
		err := rec.Add(share, sigs[i])
		if err != nil {
			t.Fatal(err)
		}
	}

	recovered, err := rec.Reconstruct()
	if err != nil {
		t.Fatal(err)
	}

	ok, err := groupKey.Verify(recovered, msg)
	if err != nil {
		t.Fatal(err)
	}

	if !ok {
		t.Error("Reconstructed signature did not verify")
	}
}

func TestNewBLS0ChainThresholdScheme(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *BLS0ChainThresholdScheme
	}{
		{
			name: "Test_NewBLS0ChainThresholdScheme_OK",
			want: &BLS0ChainThresholdScheme{},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewBLS0ChainThresholdScheme(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBLS0ChainThresholdScheme() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0ChainThresholdScheme_SetID(t *testing.T) {
	t.Parallel()

	s := NewBLS0ChainThresholdScheme()
	id := "1"
	if err := s.id.SetHexString(id); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		BLS0ChainScheme BLS0ChainScheme
		id              bls.ID
	}
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainThresholdScheme_SetID_OK",
			args:    args{id: id},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tss := &BLS0ChainThresholdScheme{
				BLS0ChainScheme: tt.fields.BLS0ChainScheme,
				id:              tt.fields.id,
			}
			if err := tss.SetID(tt.args.id); (err != nil) != tt.wantErr {
				t.Errorf("SetID() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBLS0ChainThresholdScheme_GetID(t *testing.T) {
	t.Parallel()

	type fields struct {
		BLS0ChainScheme BLS0ChainScheme
		id              bls.ID
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_BLS0ChainThresholdScheme_GetID_OK",
			want: "0",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tss := &BLS0ChainThresholdScheme{
				BLS0ChainScheme: tt.fields.BLS0ChainScheme,
				id:              tt.fields.id,
			}
			if got := tss.GetID(); got != tt.want {
				t.Errorf("GetID() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0GenerateThresholdKeyShares(t *testing.T) {
	t.Parallel()

	type args struct {
		t           int
		n           int
		originalKey SignatureScheme
	}
	tests := []struct {
		name    string
		args    args
		want    []ThresholdSignatureScheme
		wantErr bool
	}{
		{
			name:    "Test_BLS0GenerateThresholdKeyShares_Invalid_Signature_Scheme_ERR",
			args:    args{originalKey: NewED25519Scheme()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := BLS0GenerateThresholdKeyShares(tt.args.t, tt.args.n, tt.args.originalKey)
			if (err != nil) != tt.wantErr {
				t.Errorf("BLS0GenerateThresholdKeyShares() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("BLS0GenerateThresholdKeyShares() got = %v, want %v", got, tt.want)
			}
		})
	}
}

//func TestBLS0ChainReconstruction_Add(t *testing.T) {
//	t.Parallel()
//
//	type fields struct {
//		t    int
//		n    int
//		ids  []bls.ID
//		sigs []bls.Sign
//	}
//	type args struct {
//		tss       ThresholdSignatureScheme
//		signature string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name:    "Test_BLS0ChainReconstruction_Add_Invalid_Signature_Scheme_ERR",
//			args:    args{tss: &mocks.ThresholdSignatureScheme{}},
//			wantErr: true,
//		},
//		{
//			name:    "Test_BLS0ChainReconstruction_Add_Invalid_Signature_ERR",
//			args:    args{tss: NewBLS0ChainThresholdScheme(), signature: "!"},
//			wantErr: true,
//		},
//	}
//	for _, tt := range tests {
//		tt := tt
//		t.Run(tt.name, func(t *testing.T) {
//			t.Parallel()
//
//			rec := &BLS0ChainReconstruction{
//				t:    tt.fields.t,
//				n:    tt.fields.n,
//				ids:  tt.fields.ids,
//				sigs: tt.fields.sigs,
//			}
//			if err := rec.Add(tt.args.tss, tt.args.signature); (err != nil) != tt.wantErr {
//				t.Errorf("Add() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

func TestBLS0ChainReconstruction_Reconstruct(t *testing.T) {
	t.Parallel()

	type fields struct {
		t    int
		n    int
		ids  []bls.ID
		sigs []bls.Sign
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainReconstruction_Reconstruct_ERR",
			fields:  fields{ids: make([]bls.ID, 1), sigs: make([]bls.Sign, 0)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rec := BLS0ChainReconstruction{
				t:    tt.fields.t,
				n:    tt.fields.n,
				ids:  tt.fields.ids,
				sigs: tt.fields.sigs,
			}
			got, err := rec.Reconstruct()
			if (err != nil) != tt.wantErr {
				t.Errorf("Reconstruct() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Reconstruct() got = %v, want %v", got, tt.want)
			}
		})
	}
}
