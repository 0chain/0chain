package block

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"sort"
	"sync"
	"testing"

	"0chain.net/core/encryption"
	"github.com/0chain/common/core/util"
)

func TestGroupSharesOrSigns_Get(t *testing.T) {
	key := "key"
	sos := NewShareOrSigns()
	shares := map[string]*ShareOrSigns{
		key: sos,
	}

	type fields struct {
		Shares map[string]*ShareOrSigns
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ShareOrSigns
		want1  bool
	}{
		{
			name:   "TRUE",
			fields: fields{Shares: shares},
			args:   args{id: key},
			want:   sos,
			want1:  true,
		},
		{
			name:   "FALSE",
			fields: fields{Shares: shares},
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gsos := &GroupSharesOrSigns{
				mutex:  sync.RWMutex{},
				Shares: tt.fields.Shares,
			}
			got, got1 := gsos.Get(tt.args.id)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Get() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Get() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestGroupSharesOrSigns_GetShares(t *testing.T) {
	shares := map[string]*ShareOrSigns{
		"key": NewShareOrSigns(),
	}

	type fields struct {
		Shares map[string]*ShareOrSigns
	}
	tests := []struct {
		name   string
		fields fields
		want   map[string]*ShareOrSigns
	}{
		{
			name:   "OK",
			fields: fields{Shares: shares},
			want:   shares,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gsos := &GroupSharesOrSigns{
				mutex:  sync.RWMutex{},
				Shares: tt.fields.Shares,
			}
			if got := gsos.GetShares(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetShares() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupSharesOrSigns_Encode(t *testing.T) {
	gsos := NewGroupSharesOrSigns()
	gsos.Shares = map[string]*ShareOrSigns{
		"key": NewShareOrSigns(),
	}
	blob, err := json.Marshal(gsos)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Shares map[string]*ShareOrSigns
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name:   "OK",
			fields: fields{Shares: gsos.Shares},
			want:   blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gsos := &GroupSharesOrSigns{
				mutex:  sync.RWMutex{},
				Shares: tt.fields.Shares,
			}
			if got := gsos.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGroupSharesOrSigns_Decode(t *testing.T) {
	gsos := NewGroupSharesOrSigns()
	gsos.Shares = map[string]*ShareOrSigns{
		"key": NewShareOrSigns(),
	}
	blob, err := json.Marshal(gsos)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		Shares map[string]*ShareOrSigns
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *GroupSharesOrSigns
		wantErr bool
	}{
		{
			name:    "OK",
			fields:  fields{Shares: gsos.Shares},
			args:    args{input: blob},
			want:    gsos,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gsos := &GroupSharesOrSigns{
				mutex:  sync.RWMutex{},
				Shares: tt.fields.Shares,
			}
			if err := gsos.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(gsos, tt.want) {
				t.Errorf("Decode() got = %v, want = %v", gsos, tt.want)
			}
		})
	}
}

func TestGroupSharesOrSigns_GetHash(t *testing.T) {
	gsos := NewGroupSharesOrSigns()
	gsos.Shares = map[string]*ShareOrSigns{
		"key": NewShareOrSigns(),
	}

	var data []byte
	var keys []string
	for k := range gsos.Shares {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		bytes, _ := hex.DecodeString(gsos.Shares[k].Hash())
		data = append(data, bytes...)
	}
	hash := encryption.RawHash(data)

	type fields struct {
		Shares map[string]*ShareOrSigns
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "OK",
			fields: fields{
				Shares: gsos.Shares,
			},
			want: util.ToHex(hash),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gsos := &GroupSharesOrSigns{
				mutex:  sync.RWMutex{},
				Shares: tt.fields.Shares,
			}
			if got := gsos.GetHash(); got != tt.want {
				t.Errorf("GetHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
