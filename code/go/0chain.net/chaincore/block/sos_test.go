package block

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"testing"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
)

func TestNewShareOrSigns(t *testing.T) {
	tests := []struct {
		name string
		want *ShareOrSigns
	}{
		{
			name: "OK",
			want: &ShareOrSigns{ShareOrSigns: make(map[string]*bls.DKGKeyShare)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewShareOrSigns(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewShareOrSigns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShareOrSigns_Hash(t *testing.T) {
	sos := NewShareOrSigns()
	sos.ID = encryption.Hash("data")
	sos.ShareOrSigns = map[string]*bls.DKGKeyShare{
		"key": {},
	}

	data := sos.ID
	var keys []string
	for k := range sos.ShareOrSigns {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		data += string(sos.ShareOrSigns[k].Encode())
	}

	type fields struct {
		ID           string
		ShareOrSigns map[string]*bls.DKGKeyShare
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				ID:           sos.ID,
				ShareOrSigns: sos.ShareOrSigns,
			},
			want: encryption.Hash(data),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sos := &ShareOrSigns{
				ID:           tt.fields.ID,
				ShareOrSigns: tt.fields.ShareOrSigns,
			}
			if got := sos.Hash(); got != tt.want {
				t.Errorf("Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShareOrSigns_Validate(t *testing.T) {
	msg := hex.EncodeToString([]byte("message"))

	pbK, prK, err := encryption.GenerateKeys()
	if err != nil {
		t.Fatal(err)
	}
	sign, err := encryption.Sign(prK, msg)
	if err != nil {
		t.Fatal(err)
	}

	sos := NewShareOrSigns()
	sos.ID = encryption.Hash("data")
	key := pbK
	sos.ShareOrSigns = map[string]*bls.DKGKeyShare{
		key: {
			Sign:    sign,
			Message: msg,
		},
	}

	type fields struct {
		ID           string
		ShareOrSigns map[string]*bls.DKGKeyShare
	}
	type args struct {
		mpks       *Mpks
		publicKeys map[string]string
		scheme     encryption.SignatureScheme
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []string
		want1  bool
	}{
		{
			name:   "Unknown_Public_Key_FALSE",
			fields: fields{ID: sos.ID, ShareOrSigns: sos.ShareOrSigns},
			want1:  false,
		},
		{
			name: "Invalid_Sign_FALSE",
			fields: fields{
				ID: sos.ID,
				ShareOrSigns: map[string]*bls.DKGKeyShare{
					key: {
						Sign: sign,
					},
				},
			},
			args: args{
				publicKeys: map[string]string{
					key: pbK,
				},
				scheme: &encryption.ED25519Scheme{},
			},
			want1: false,
		},
		{
			name: "Set_Hex_String_Err_FALSE",
			fields: fields{
				ID: sos.ID,
				ShareOrSigns: map[string]*bls.DKGKeyShare{
					key: {
						Share: encryption.Hash("share"),
					},
				},
			},
			args: args{
				mpks: &Mpks{
					Mpks: map[string]*MPK{
						sos.ID: {
							ID: "id",
							Mpk: []string{
								"mpk",
							},
						},
					},
				},
				publicKeys: map[string]string{
					key: pbK,
				},
				scheme: &encryption.ED25519Scheme{},
			},
			want1: false,
		},
		{
			name: "TRUE",
			fields: fields{
				ID:           sos.ID,
				ShareOrSigns: map[string]*bls.DKGKeyShare{},
			},
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sos := &ShareOrSigns{
				ID:           tt.fields.ID,
				ShareOrSigns: tt.fields.ShareOrSigns,
			}
			got, got1 := sos.Validate(tt.args.mpks, tt.args.publicKeys, tt.args.scheme)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Validate() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Validate() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestShareOrSigns_Encode(t *testing.T) {
	sos := NewShareOrSigns()
	sos.ID = encryption.Hash("data")
	sos.ShareOrSigns = map[string]*bls.DKGKeyShare{
		"key": {
			Sign:    "sign",
			Message: "msg",
		},
	}
	blob, err := json.Marshal(sos)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ID           string
		ShareOrSigns map[string]*bls.DKGKeyShare
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				ID:           sos.ID,
				ShareOrSigns: sos.ShareOrSigns,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sos := &ShareOrSigns{
				ID:           tt.fields.ID,
				ShareOrSigns: tt.fields.ShareOrSigns,
			}
			if got := sos.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShareOrSigns_Decode(t *testing.T) {
	sos := NewShareOrSigns()
	sos.ID = encryption.Hash("data")
	sos.ShareOrSigns = map[string]*bls.DKGKeyShare{
		"key": {
			Sign:    "sign",
			Message: "msg",
		},
	}
	blob, err := json.Marshal(sos)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ID           string
		ShareOrSigns map[string]*bls.DKGKeyShare
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *ShareOrSigns
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{input: blob},
			want:    sos,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sos := &ShareOrSigns{
				ID:           tt.fields.ID,
				ShareOrSigns: tt.fields.ShareOrSigns,
			}
			if err := sos.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(sos, tt.want) {
				t.Errorf("Decode() got = %v, want = %v", sos, tt.want)
			}
		})
	}
}

func TestShareOrSigns_Clone(t *testing.T) {
	sos := NewShareOrSigns()
	sos.ID = encryption.Hash("data")
	sos.ShareOrSigns = map[string]*bls.DKGKeyShare{
		"key": {
			Sign:    "sign",
			Message: "msg",
		},
	}

	type fields struct {
		ID           string
		ShareOrSigns map[string]*bls.DKGKeyShare
	}
	tests := []struct {
		name   string
		fields fields
		want   *ShareOrSigns
	}{
		{
			name: "OK",
			fields: fields{
				ID:           sos.ID,
				ShareOrSigns: sos.ShareOrSigns,
			},
			want: sos,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sos := &ShareOrSigns{
				ID:           tt.fields.ID,
				ShareOrSigns: tt.fields.ShareOrSigns,
			}
			if got := sos.Clone(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Clone() = %v, want %v", got, tt.want)
			}
		})
	}
}
