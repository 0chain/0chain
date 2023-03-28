package encryption

import (
	"bytes"
	"encoding/hex"
	"io"
	"reflect"
	"testing"

	"github.com/herumi/bls/ffi/go/bls"
	"github.com/herumi/mcl/ffi/go/mcl"
	"github.com/stretchr/testify/require"
)

func TestMiraclToHerumiPK(t *testing.T) {
	miraclpk1 := `0418a02c6bd223ae0dfda1d2f9a3c81726ab436ce5e9d17c531ff0a385a13a0b491bdfed3a85690775ee35c61678957aaba7b1a1899438829f1dc94248d87ed36817f6dfafec19bfa87bf791a4d694f43fec227ae6f5a867490e30328cac05eaff039ac7dfc3364e851ebd2631ea6f1685609fc66d50223cc696cb59ff2fee47ac`
	pk1 := MiraclToHerumiPK(miraclpk1)

	require.EqualValues(t, pk1, "68d37ed84842c91d9f82389489a1b1a7ab7a957816c635ee750769853aeddf1b490b3aa185a3f01f537cd1e9e56c43ab2617c8a3f9d2a1fd0dae23d26b2ca018")

	// Assert DeserializeHexStr works on the output of MiraclToHerumiPK
	var pk bls.PublicKey
	err := pk.DeserializeHexStr(pk1)
	require.NoError(t, err)
}

func TestMiraclToHerumiSig(t *testing.T) {
	miraclsig1 := `(0d4dbad6d2586d5e01b6b7fbad77e4adfa81212c52b4a0b885e19c58e0944764,110061aa16d5ba36eef0ad4503be346908d3513c0a2aedfd0d2923411b420eca)`
	sig1 := MiraclToHerumiSig(miraclsig1)

	// Assert DeserializeHexStr works on the output of MiraclToHerumiSig
	var sig bls.Sign
	err := sig.DeserializeHexStr(sig1)
	require.NoError(t, err)

	// Test that passing in normal herumi sig just gets back the original.
	sig2 := MiraclToHerumiSig(sig1)
	if sig1 != sig2 {
		panic("Sigs should've been the same")
	}
}

func TestBLS0ChainGenerateKeys(t *testing.T) {
	b0scheme := NewBLS0ChainScheme()
	err := b0scheme.GenerateKeys()
	require.NoError(t, err)
}

func TestBLS0ChainWriteKeys(t *testing.T) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
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
	err := sigScheme.GenerateKeys()
	require.NoError(t, err)
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		panic(err)
	}
	if ok, err := sigScheme.Verify(signature, expectedHash); err != nil || !ok {
		t.Errorf("Verification failed\n")
	}
}

func BenchmarkBLS0ChainSign(b *testing.B) {
	sigScheme := NewBLS0ChainScheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		_, err := sigScheme.Sign(expectedHash)
		require.NoError(b, err)
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
		_, err := sigScheme.PairMessageHash(expectedHash)
		require.NoError(b, err)
	}
}

func BenchmarkBLS0ChainG1HashToPoint(b *testing.B) {
	var g1 mcl.G1
	rawHash := RawHash("bls-0chain-signature-scheme")
	for i := 0; i < b.N; i++ {
		err := g1.HashAndMapTo(rawHash)
		require.NoError(b, err)
	}
}

func TestBLS0ChainScheme_ReadKeys(t *testing.T) {
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

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
	t.Parallel()

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
			name:    "TestBLS0ChainScheme_PairMessageHash_Hex_Decoding_Hash_ERR",
			fields:  fields{publicKey: scheme.publicKey},
			args:    args{hash: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			b0, err := newBLS0ChainSchemeFromPublicKey(tt.fields.publicKey)
			require.NoError(t, err)

			_, err = b0.PairMessageHash(tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("PairMessageHash() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
