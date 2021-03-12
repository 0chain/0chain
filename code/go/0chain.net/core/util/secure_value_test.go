package util

import (
	"0chain.net/core/encryption"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"
)

func TestHashStringToBytes(t *testing.T) {
	t.Parallel()

	enc := make([]byte, 0)
	hex.Encode([]byte("data"), enc)
	dec, err := hex.DecodeString(string(enc))
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		hash string
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "Test_HashStringToBytes_OK",
			args: args{hash: string(enc)},
			want: dec,
		},
		{
			name: "Test_HashStringToBytes_ERR",
			args: args{hash: "!"},
			want: nil,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := HashStringToBytes(tt.args.hash); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("HashStringToBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecureSerializableValue_GetHash(t *testing.T) {
	t.Parallel()

	var (
		buff = []byte(encryption.Hash("data"))
		want = hex.EncodeToString(encryption.RawHash(buff))
	)

	type fields struct {
		Buffer []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name:   "Test_SecureSerializableValue_GetHash_OK",
			fields: fields{Buffer: buff},
			want:   want,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spv := &SecureSerializableValue{
				Buffer: tt.fields.Buffer,
			}
			if got := spv.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestToUpperHex(t *testing.T) {
	t.Parallel()

	buf := []byte("data")

	type args struct {
		buf []byte
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_ToUpperHex_OK",
			args: args{buf: buf},
			want: strings.ToUpper(hex.EncodeToString(buf)),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := ToUpperHex(tt.args.buf); got != tt.want {
				t.Errorf("ToUpperHex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecureSerializableValue_Encode(t *testing.T) {
	t.Parallel()

	buf := []byte("data")

	type fields struct {
		Buffer []byte
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "Test_SecureSerializableValue_Encode_OK",
			fields: fields{Buffer: buf},
			want:   buf,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spv := &SecureSerializableValue{
				Buffer: tt.fields.Buffer,
			}
			if got := spv.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecureSerializableValue_Decode(t *testing.T) {
	t.Parallel()

	buf := []byte("data")

	type fields struct {
		Buffer []byte
	}
	type args struct {
		buf []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *SecureSerializableValue
		wantErr bool
	}{
		{
			name:   "Test_SecureSerializableValue_Decode_OK",
			fields: fields{},
			args:   args{buf: buf},
			want:   &SecureSerializableValue{Buffer: buf},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spv := &SecureSerializableValue{
				Buffer: tt.fields.Buffer,
			}
			if err := spv.Decode(tt.args.buf); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
