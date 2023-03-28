package block

import (
	"encoding/json"
	"reflect"
	"testing"

	"0chain.net/chaincore/threshold/bls"
	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
	"github.com/stretchr/testify/require"
)

func TestNewMpks(t *testing.T) {
	tests := []struct {
		name string
		want *Mpks
	}{
		{
			name: "OK",
			want: &Mpks{Mpks: make(map[string]*MPK)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMpks(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMpks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMpks_Encode(t *testing.T) {
	mpk := NewMpks()
	mpk.Mpks["key"] = &MPK{ID: "id"}
	blob, err := json.Marshal(mpk)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Mpks map[string]*MPK
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				Mpks: mpk.Mpks,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpks := &Mpks{
				Mpks: tt.fields.Mpks,
			}
			if got := mpks.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMpks_Decode(t *testing.T) {
	t.Parallel()

	mpk := NewMpks()
	mpk.Mpks["key"] = &MPK{ID: "id"}
	blob, err := json.Marshal(mpk)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Mpks map[string]*MPK
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Mpks
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				Mpks: map[string]*MPK{
					"key": {ID: "id"},
				},
			},
			args: args{
				input: func() []byte {
					res := make([]byte, len(blob))
					copy(res, blob)
					return res
				}(),
			},
			want: mpk,
		},
		{
			name: "ERR",
			fields: fields{
				Mpks: map[string]*MPK{
					"key": {ID: "id"},
				},
			},
			args:    args{input: []byte("}{")},
			wantErr: true,
		},
		// duplicating tests to expose race errors
		{
			name: "OK",
			fields: fields{
				Mpks: map[string]*MPK{
					"key": {ID: "id"},
				},
			},
			args: args{
				input: func() []byte {
					res := make([]byte, len(blob))
					copy(res, blob)
					return res
				}(),
			},
			want: mpk,
		},
		{
			name: "ERR",
			fields: fields{
				Mpks: map[string]*MPK{
					"key": {ID: "id"},
				},
			},
			args:    args{input: []byte("}{")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mpks := &Mpks{
				Mpks: tt.fields.Mpks,
			}
			if err := mpks.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(mpks, tt.want) {
				t.Errorf("Decode() got = %v, want = %v", mpks, tt.want)
			}
		})
	}
}

func TestMpks_GetHash(t *testing.T) {
	mpk := NewMpks()
	mpk.Mpks["key"] = &MPK{ID: "id"}

	type fields struct {
		Mpks map[string]*MPK
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				Mpks: mpk.Mpks,
			},
			want: util.ToHex(encryption.RawHash(mpk.Encode())),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpks := &Mpks{
				Mpks: tt.fields.Mpks,
			}
			if got := mpks.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMpks_GetMpkMap(t *testing.T) {
	mpk := NewMpks()
	mpk.Mpks[encryption.Hash("key")] = &MPK{ID: encryption.Hash("data")}

	mpkMap := make(map[bls.PartyID][]bls.PublicKey)
	for k, v := range mpk.Mpks {
		mpks, err := bls.ConvertStringToMpk(v.Mpk)
		require.NoError(t, err)
		mpkMap[bls.ComputeIDdkg(k)] = mpks
	}

	type fields struct {
		Mpks map[string]*MPK
	}
	tests := []struct {
		name   string
		fields fields
		want   map[bls.PartyID][]bls.PublicKey
	}{
		{
			name:   "OK",
			fields: fields{Mpks: mpk.Mpks},
			want:   mpkMap,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpks := &Mpks{
				Mpks: tt.fields.Mpks,
			}
			got, err := mpks.GetMpkMap()
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMpks_GetMpks(t *testing.T) {
	mpk := NewMpks()
	mpk.Mpks["key"] = &MPK{ID: "id"}

	type fields struct {
		Mpks map[string]*MPK
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]*MPK
	}{
		{
			name:   "OK",
			fields: fields{Mpks: mpk.Mpks},
			want:   mpk.Mpks,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpks := &Mpks{
				Mpks: tt.fields.Mpks,
			}
			if got := mpks.GetMpks(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetMpks() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMPK_Encode(t *testing.T) {
	mpk := &MPK{ID: "id"}
	blob, err := json.Marshal(mpk)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ID  string
		Mpk []string
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				ID:  mpk.ID,
				Mpk: mpk.Mpk,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpk := &MPK{
				ID:  tt.fields.ID,
				Mpk: tt.fields.Mpk,
			}
			if got := mpk.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMPK_Decode(t *testing.T) {
	mpk := &MPK{ID: "id"}
	blob, err := json.Marshal(mpk)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		ID  string
		Mpk []string
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *MPK
		wantErr bool
	}{
		{
			name: "OK",
			fields: fields{
				ID:  mpk.ID,
				Mpk: mpk.Mpk,
			},
			args:    args{input: blob},
			want:    mpk,
			wantErr: false,
		},
		{
			name: "ERR",
			fields: fields{
				ID:  mpk.ID,
				Mpk: mpk.Mpk,
			},
			args:    args{input: []byte("}{")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mpk := &MPK{
				ID:  tt.fields.ID,
				Mpk: tt.fields.Mpk,
			}
			if err := mpk.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(mpk, tt.want) {
				t.Errorf("Decode() got = %v, want = %v", mpk, tt.want)
			}
		})
	}
}
