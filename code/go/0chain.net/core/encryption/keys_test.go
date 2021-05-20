package encryption

import (
	"bytes"
	"io"
	"testing"
)

func TestReadKeys(t *testing.T) {
	t.Parallel()

	pbK, prK, err := GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name           string
		args           args
		wantSucces     bool
		wantPublicKey  string
		wantPrivateKey string
	}{
		{
			name:       "Test_ReadKeys_Empty_First_Line_FALSE",
			args:       args{reader: bytes.NewBuffer(nil)},
			wantSucces: false,
		},
		{
			name:          "Test_ReadKeys_Empty_Second_Line_FALSE",
			args:          args{reader: bytes.NewBuffer([]byte(pbK + "\n"))},
			wantSucces:    false,
			wantPublicKey: pbK,
		},
		{
			name:           "Test_ReadKeys_Empty_TRUE",
			args:           args{reader: bytes.NewBuffer([]byte(pbK + "\n" + prK))},
			wantSucces:     true,
			wantPrivateKey: prK,
			wantPublicKey:  pbK,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotSucces, gotPublicKey, gotPrivateKey := ReadKeys(tt.args.reader)
			if gotSucces != tt.wantSucces {
				t.Errorf("ReadKeys() gotSucces = %v, want %v", gotSucces, tt.wantSucces)
			}
			if gotPublicKey != tt.wantPublicKey {
				t.Errorf("ReadKeys() gotPublicKey = %v, want %v", gotPublicKey, tt.wantPublicKey)
			}
			if gotPrivateKey != tt.wantPrivateKey {
				t.Errorf("ReadKeys() gotPrivateKey = %v, want %v", gotPrivateKey, tt.wantPrivateKey)
			}
		})
	}
}

func TestVerify(t *testing.T) {
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

	type args struct {
		publicKey interface{}
		signature string
		hash      string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "Test_Verify_OK",
			args: args{signature: sign, publicKey: pbK, hash: hash},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Verify(tt.args.publicKey, tt.args.signature, tt.args.hash)
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
