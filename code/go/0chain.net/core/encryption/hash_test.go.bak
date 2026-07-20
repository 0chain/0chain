package encryption

import (
	"encoding/hex"
	"reflect"
	"testing"

	"golang.org/x/crypto/sha3"
)

func TestRawHash(t *testing.T) {
	t.Parallel()

	data := []byte("data")

	hb := HashBytes{}
	copy(hb[:], data)

	type args struct {
		data interface{}
	}
	tests := []struct {
		name      string
		args      args
		want      []byte
		wantPanic bool
	}{
		{
			name: "Test_RawHash_Bytes_OK",
			args: args{data: data},
			want: func() []byte {
				hash := sha3.New256()
				hash.Write(data)
				var buf []byte
				return hash.Sum(buf)
			}(),
		},
		{
			name: "Test_RawHash_Hash_Bytes_OK",
			args: args{data: hb},
			want: func() []byte {
				hash := sha3.New256()
				hash.Write(hb[:])
				var buf []byte
				return hash.Sum(buf)
			}(),
		},
		{
			name: "Test_RawHash_String_OK",
			args: args{data: string(data)},
			want: func() []byte {
				hash := sha3.New256()
				hash.Write(data)
				var buf []byte
				return hash.Sum(buf)
			}(),
		},
		{
			name:      "Test_RawHash_Panic",
			args:      args{data: 123},
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
					t.Errorf("RawHash() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := RawHash(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RawHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHash(t *testing.T) {
	t.Parallel()

	data := "data"

	type args struct {
		data interface{}
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test_Hash_OK",
			args: args{data: data},
			want: hex.EncodeToString(RawHash(data)),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Hash(tt.args.data); got != tt.want {
				t.Errorf("Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsHash(t *testing.T) {
	t.Parallel()

	type args struct {
		str string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_IsHash_TRUE",
			args: args{str: Hash("data")},
			want: true,
		},
		{
			name: "Test_IsHash_False",
			args: args{str: "not hash"},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := IsHash(tt.args.str); got != tt.want {
				t.Errorf("IsHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
