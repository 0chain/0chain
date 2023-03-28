package encryption

import (
	"bytes"
	"encoding/hex"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/ed25519"
)

var expectedHash = "6cb51770083ba34e046bc6c953f9f05b64e16a0956d4e496758b97c9cf5687d5"

func TestED25519GenerateKeys(t *testing.T) {
	sigScheme := NewED25519Scheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
}

func TestED25519ChainWriteKeys(t *testing.T) {
	sigScheme := NewED25519Scheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
}

func TestED25519ReadKeys(t *testing.T) {
	reader := bytes.NewBuffer([]byte("e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0\naa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"))
	sigScheme := NewED25519Scheme()
	err := sigScheme.ReadKeys(reader)
	if err != nil {
		t.Fatal(err)
	}
}

func TestED25519SignAndVerify(t *testing.T) {
	sigScheme := NewED25519Scheme()
	buffer := bytes.NewBuffer([]byte("e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0\naa3e1ae2290987959dc44e43d138c81f15f93b2d56d7a06c51465f345df1a8a6e065fc02aaf7aaafaebe5d2dedb9c7c1d63517534644434b813cb3bdab0f94a0"))
	err := sigScheme.ReadKeys(buffer)
	require.NoError(t, err)
	signature, err := sigScheme.Sign(expectedHash)
	if err != nil {
		t.Fatal(err)
	}
	if ok, err := sigScheme.Verify(signature, expectedHash); err != nil || !ok {
		t.Error("Verification failed\n")
	}
}

func BenchmarkED25519GenerateKeys(b *testing.B) {
	sigScheme := NewED25519Scheme()
	for i := 0; i < b.N; i++ {
		err := sigScheme.GenerateKeys()
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkED25519Sign(b *testing.B) {
	sigScheme := NewED25519Scheme()
	err := sigScheme.GenerateKeys()
	if err != nil {
		panic(err)
	}
	for i := 0; i < b.N; i++ {
		_, err := sigScheme.Sign(expectedHash)
		require.NoError(b, err)
	}
}

func BenchmarkED25519Verify(b *testing.B) {
	sigScheme := NewED25519Scheme()
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

func TestED25519Scheme_ReadKeys(t *testing.T) {
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
			name:    "Test_ED25519Scheme_ReadKeys_First_Line_Reading_ERR",
			args:    args{reader: bytes.NewBuffer(nil)},
			wantErr: true,
		},
		{
			name:    "Test_ED25519Scheme_ReadKeys_Second_Line_Reading_ERR",
			args:    args{reader: bytes.NewBuffer([]byte(pbK))},
			wantErr: true,
		},
		{
			name:    "Test_ED25519Scheme_ReadKeys_Hex_Decoding_ERR",
			args:    args{reader: bytes.NewBuffer([]byte(pbK + "\n" + prK + "!"))},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ed := &ED25519Scheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			if err := ed.ReadKeys(tt.args.reader); (err != nil) != tt.wantErr {
				t.Errorf("ReadKeys() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestED25519Scheme_SetPublicKey(t *testing.T) {
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
			name: "Test_ED25519Scheme_SetPublicKey_Setted_Private_Key_ERR",
			fields: fields{
				privateKey: []byte("pr key"),
			},
			wantErr: true,
		},
		{
			name:    "Test_ED25519Scheme_SetPublicKey_Hex_Decoding_ERR",
			args:    args{publicKey: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ed := &ED25519Scheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			if err := ed.SetPublicKey(tt.args.publicKey); (err != nil) != tt.wantErr {
				t.Errorf("SetPublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestED25519Scheme_GetPublicKey(t *testing.T) {
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
			name:   "Test_ED25519Scheme_GetPublicKey_OK",
			fields: fields{publicKey: b},
			want:   pbK,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ed := &ED25519Scheme{
				privateKey: tt.fields.privateKey,
				publicKey:  tt.fields.publicKey,
			}
			if got := ed.GetPublicKey(); got != tt.want {
				t.Errorf("GetPublicKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_signED25519(t *testing.T) {
	t.Parallel()

	_, prK, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	hash := Hash("data")
	rawHash, err := GetRawHash(hash)
	if err != nil {
		t.Fatal(err)
	}

	b, err := hex.DecodeString(prK)
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		privateKey interface{}
		hash       interface{}
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name:    "Test_signED25519_String_Key_OK",
			args:    args{privateKey: prK, hash: hash},
			want:    hex.EncodeToString(ed25519.Sign(b, rawHash)),
			wantErr: false,
		},
		{
			name:    "Test_signED25519_String_Key_Invalid_Private_Key_ERR",
			args:    args{privateKey: "!"},
			wantErr: true,
		},
		{
			name:    "Test_signED25519_String_Key_Invalid_Hash_ERR",
			args:    args{privateKey: prK, hash: "!"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := signED25519(tt.args.privateKey, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("signED25519() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("signED25519() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_verifyED25519(t *testing.T) {
	t.Parallel()

	pbK, prK, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	hash := Hash("data")
	sign, err := Sign(prK, hash)
	if err != nil {
		t.Fatal(err)
	}

	pbKHashB := HashBytes{}
	pbKBytes, err := hex.DecodeString(pbK)
	if err != nil {
		t.Fatal(err)
	}
	copy(pbKHashB[:], pbKBytes)

	type args struct {
		publicKey interface{}
		signature string
		hash      string
	}
	tests := []struct {
		name      string
		args      args
		want      bool
		wantErr   bool
		wantPanic bool
	}{
		{
			name: "Test_verifyED25519_Hash_Bytes_Public_Key_OK",
			args: args{
				publicKey: pbKHashB,
				signature: sign,
				hash:      hash,
			},
			want: true,
		},
		{
			name: "Test_verifyED25519_Decoding_Sign_OK",
			args: args{
				publicKey: pbKHashB,
				signature: "!",
				hash:      hash,
			},
			wantErr: true,
		},
		{
			name: "Test_verifyED25519_Decoding_Public_Key_ERR",
			args: args{
				publicKey: "!",
				signature: sign,
				hash:      hash,
			},
			wantErr: true,
		},
		{
			name: "Test_verifyED25519_Decoding_Hash_ERR",
			args: args{
				publicKey: pbK,
				signature: sign,
				hash:      "!",
			},
			wantErr: true,
		},
		{
			name: "Test_verifyED25519_PANIC",
			args: args{
				publicKey: 123,
				signature: sign,
				hash:      hash,
			},
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
					t.Errorf("verifyED25519() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got, err := verifyED25519(tt.args.publicKey, tt.args.signature, tt.args.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifyED25519() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("verifyED25519() got = %v, want %v", got, tt.want)
			}
		})
	}
}
