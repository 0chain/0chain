package state

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"0chain.net/core/encryption"
)

func TestSignedTransfer_Sign(t *testing.T) {
	t.Parallel()

	st := SignedTransfer{
		Transfer: *NewTransfer("from client id", "to client id", 5),
	}
	scheme := encryption.NewED25519Scheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}

	sign, err := scheme.Sign(encryption.Hash(st.Transfer.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Transfer   Transfer
		SchemeName string
		PublicKey  string
		Sig        string
	}
	type args struct {
		sigScheme encryption.SignatureScheme
	}
	tests := []struct {
		name     string
		fields   fields
		args     args
		wantErr  bool
		wantSign string
	}{
		{
			name: "OK",
			fields: fields{
				Transfer: st.Transfer,
			},
			args: args{
				sigScheme: scheme,
			},
			wantErr:  false,
			wantSign: sign,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st := &SignedTransfer{
				Transfer:   tt.fields.Transfer,
				SchemeName: tt.fields.SchemeName,
				PublicKey:  tt.fields.PublicKey,
				Sig:        tt.fields.Sig,
			}
			if err := st.Sign(tt.args.sigScheme); (err != nil) != tt.wantErr {
				t.Errorf("Sign() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.Equal(t, tt.wantSign, st.Sig)
		})
	}
}

func TestSignedTransfer_VerifySignature(t *testing.T) {
	t.Parallel()

	st := SignedTransfer{
		Transfer: *NewTransfer("from client id", "to client id", 5),
	}
	scheme := encryption.NewED25519Scheme()
	if err := scheme.GenerateKeys(); err != nil {
		t.Fatal(err)
	}

	sign, err := scheme.Sign(encryption.Hash(st.Transfer.Encode()))
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Transfer   Transfer
		SchemeName string
		PublicKey  string
		Sig        string
	}
	type args struct {
		requireSendersSignature bool
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Invalid_Scheme_ERR",
			fields:  fields{SchemeName: "unknown scheme"},
			wantErr: true,
		},
		{
			name: "Verifying_Public_Key_ERR",
			fields: fields{
				SchemeName: "ed25519",
				PublicKey:  "invalid public key",
			},
			args:    args{requireSendersSignature: true},
			wantErr: true,
		},
		{
			name: "Setting_Public_Key_ERR",
			fields: fields{
				SchemeName: "ed25519",
				PublicKey:  "invalid public key",
			},
			args:    args{requireSendersSignature: false},
			wantErr: true,
		},
		{
			name: "Verifying_Signature_ERR",
			fields: fields{
				SchemeName: "ed25519",
				PublicKey:  scheme.GetPublicKey(),
			},
			args:    args{requireSendersSignature: false},
			wantErr: true,
		},
		{
			name: "Invalid_Signature_ERR",
			fields: fields{
				SchemeName: "ed25519",
				PublicKey:  scheme.GetPublicKey(),
				Sig:        "!!",
			},
			args:    args{requireSendersSignature: false},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				Transfer:   st.Transfer,
				SchemeName: "ed25519",
				PublicKey:  scheme.GetPublicKey(),
				Sig:        sign,
			},
			args:    args{requireSendersSignature: false},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st := SignedTransfer{
				Transfer:   tt.fields.Transfer,
				SchemeName: tt.fields.SchemeName,
				PublicKey:  tt.fields.PublicKey,
				Sig:        tt.fields.Sig,
			}
			if err := st.VerifySignature(tt.args.requireSendersSignature); (err != nil) != tt.wantErr {
				t.Errorf("VerifySignature() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSignedTransfer_verifyPublicKey(t *testing.T) {
	t.Parallel()

	pbK, _, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Transfer   Transfer
		SchemeName string
		PublicKey  string
		Sig        string
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "Invalid_Public_Key_ERR",
			fields: fields{
				PublicKey: "!",
			},
			wantErr: true,
		},
		{
			name: "Wrong_Public_Key_ERR",
			fields: fields{
				PublicKey: pbK,
			},
			wantErr: true,
		},
		{
			name: "OK",
			fields: fields{
				Transfer: Transfer{
					ClientID: func() string {
						clientID, err := hex.DecodeString(pbK)
						if err != nil {
							t.Fatal(err)
						}

						return encryption.Hash(clientID)
					}(),
				},
				PublicKey: pbK,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st := SignedTransfer{
				Transfer:   tt.fields.Transfer,
				SchemeName: tt.fields.SchemeName,
				PublicKey:  tt.fields.PublicKey,
				Sig:        tt.fields.Sig,
			}
			if err := st.verifyPublicKey(); (err != nil) != tt.wantErr {
				t.Errorf("verifyPublicKey() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSignedTransfer_computeTransferHash(t *testing.T) {
	t.Parallel()

	tr := NewTransfer("from client id", "to client id", 5)

	type fields struct {
		Transfer   Transfer
		SchemeName string
		PublicKey  string
		Sig        string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				Transfer: *tr,
			},
			want: encryption.Hash(tr.Encode()),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			st := SignedTransfer{
				Transfer:   tt.fields.Transfer,
				SchemeName: tt.fields.SchemeName,
				PublicKey:  tt.fields.PublicKey,
				Sig:        tt.fields.Sig,
			}
			if got := st.computeTransferHash(); got != tt.want {
				t.Errorf("computeTransferHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
