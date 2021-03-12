package encryption

import (
	"0chain.net/chaincore/threshold/bls"
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"reflect"
	"testing"

	"github.com/herumi/mcl/ffi/go/mcl"
)

func TestBLS0ChainGenerateKeys(t *testing.T) {
	b0scheme := NewBLS0ChainScheme()
	b0scheme.GenerateKeys()
}

func TestBLS0ChainWriteKeys(t *testing.T) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	sigScheme.WriteKeys(os.Stdout)
}

func TestBLS0ChainReadKeys(t *testing.T) {
	str := `4123d01678a8b9a9cec8315241710093bb50de802ec79cdb22df28d8ced81720f7637e5db8a4f6037f89daecaff7a223caee9d71cb101107e1da024545141883
30fb9f7b7228a53f383a4647e6694646ceee0bdc015cf42bc3bbec8326302613`
	reader := bytes.NewReader([]byte(str))
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.ReadKeys(reader)
	if err != nil {
		panic(err)
	}
}

func BenchmarkBLS0ChainGenerateKeys(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	for i := 0; i < b.N; i++ {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
	}
}

func TestBLS0ChainSignAndVerify(t *testing.T) {
	sigScheme := NewBLS0ChainScheme()
	sigScheme.GenerateKeys()
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		panic(err)
	}
	fmt.Printf("signature: %T %v\n", signature, signature)
	if ok, err := sigScheme.Verify(signature, expectedHash); err != nil || !ok {
		fmt.Printf("Verification failed\n")
	} else {
		fmt.Printf("Signing Verification successful\n")
	}
}

func BenchmarkBLS0ChainSign(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.Sign(expectedHash)
	}
}

func BenchmarkBLS0ChainVerify(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		return
	}
	for i := 0; i < b.N; i++ {
		ok, err := sigScheme.Verify(signature, expectedHash)
		if err != nil {
			panic(err)
		}
		if !ok {
			panic("sig verification failed")
		}
	}
}

func BenchmarkBLS0ChainPairMessageHash(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		sigScheme.PairMessageHash(expectedHash)
	}
}

func BenchmarkBLS0ChainG1HashToPoint(b *testing.B) {
	var g1 mcl.G1
	rawHash := RawHash("bls-0chain-signature-scheme")
	for i := 0; i < b.N; i++ {
		g1.HashAndMapTo(rawHash)
	}
}

func TestBLS0ChainScheme_ReadKeys(t *testing.T) {
	pbK, prK, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainScheme_ReadKeys_First_Line_Empty_Reader_ERR",
			args:    args{reader: bytes.NewBuffer(nil)},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainScheme_ReadKeys_Second_Line_Empty_Reader_ERR",
			args:    args{reader: bytes.NewBuffer([]byte(pbK))},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainScheme_ReadKeys_Hex_Decoding_ERR",
			args:    args{reader: bytes.NewBuffer([]byte(pbK + "\n" + prK + "!"))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			if err := b0.ReadKeys(tt.args.reader); (err != nil) != tt.wantErr {
				t.Errorf("ReadKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBLS0ChainScheme_SetPublicKey(t *testing.T) {
	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	type args struct {
		publicKey string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainScheme_SetPublicKey_Setted_Key_ERR",
			fields:  fields{privateKey: []byte("private key")},
			args:    args{publicKey: "public key"},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainScheme_SetPublicKey_Hex_Decoding_ERR",
			args:    args{publicKey: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			if err := b0.SetPublicKey(tt.args.publicKey); (err != nil) != tt.wantErr {
				t.Errorf("SetPublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBLS0ChainScheme_GetPublicKey(t *testing.T) {
	pbK, _, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	b := make([]byte, hex.DecodedLen(len(pbK)))
	_, err = hex.Decode(b, []byte(pbK))
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Test_BLS0ChainScheme_GetPublicKey_OK",
			fields: fields{publicKey: b},
			want:   pbK,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			if got := b0.GetPublicKey(); got != tt.want {
				t.Errorf("GetPublicKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0ChainScheme_Sign(t *testing.T) {
	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	type args struct {
		hash interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "TestBLS0ChainScheme_Sign_Hex_Decoding_ERR",
			fields:  fields{privateKey: []byte("private key")},
			args:    args{hash: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			got, err := b0.Sign(tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Sign() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Sign() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0ChainScheme_Verify(t *testing.T) {
	scheme := NewBLS0ChainScheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}

	hash := Hash("data")
	sign, err := scheme.Sign(hash)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	type args struct {
		signature string
		hash      string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainScheme_Verify_Deserialize_ERR",
			fields:  fields{publicKey: make([]byte, 1)},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainScheme_Verify_Empty_Signature_ERR",
			fields:  fields{publicKey: scheme.publicKey},
			wantErr: true,
		},
		{
			name:    "Test_BLS0ChainScheme_Verify_Empty_Hex_Decoding_Hash_ERR",
			fields:  fields{publicKey: scheme.publicKey},
			args:    args{signature: sign, hash: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			got, err := b0.Verify(tt.args.signature, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("Verify() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Verify() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0ChainScheme_GetSignature(t *testing.T) {
	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	type args struct {
		signature string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *bls.Sign
		wantErr bool
	}{
		{
			name:    "Test_BLS0ChainScheme_GetSignature_Hex_DEcoding_ERR",
			args:    args{signature: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			got, err := b0.GetSignature(tt.args.signature)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetSignature() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetSignature() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBLS0ChainScheme_PairMessageHash(t *testing.T) {
	scheme := NewBLS0ChainScheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}

	type fields struct {
		privateKey []byte
		publicKey  []byte
	}
	type args struct {
		hash string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "TestBLS0ChainScheme_PairMessageHash_Deserializing_ERR",
			fields:  fields{publicKey: make([]byte, 1)},
			wantErr: true,
		},
		{
			name:    "TestBLS0ChainScheme_PairMessageHash_Hex_Decoding_Hash_ERR",
			fields:  fields{publicKey: scheme.publicKey},
			args:    args{hash: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b0 := &BLS0ChainScheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			_, err := b0.PairMessageHash(tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("PairMessageHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
