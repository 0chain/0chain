package block

import (
	"0chain.net/core/datastore"
	mocks "0chain.net/mocks/core/datastore"
	"context"
	"encoding/json"
	"reflect"
	"testing"
)

func TestMagicBlockMap_GetEntityMetadata(t *testing.T) {
	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "OK",
			want: magicBlockMapEntityMetadata,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if got := mb.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlockMap_GetKey(t *testing.T) {
	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Key
	}{
		{
			name: "OK",
			want: "key",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}

			mb.SetKey(tt.want)
			if got := mb.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlockMap_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(MagicBlockMap)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	magicBlockMapEntityMetadata.Store = &store

	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	type args struct {
		ctx context.Context
		key datastore.Key
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if err := mb.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMagicBlockMap_GetScore(t *testing.T) {
	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	tests := []struct {
		name   string
		fields fields
		want   int64
	}{
		{
			name: "OK",
			want: 0, // not implemented
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if got := mb.GetScore(); got != tt.want {
				t.Errorf("GetScore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlockMap_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(MagicBlockMap)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	magicBlockMapEntityMetadata.Store = &store

	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if err := mb.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMagicBlockMap_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(MagicBlockMap)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	magicBlockMapEntityMetadata.Store = &store

	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if err := mb.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMagicBlockMap_Encode(t *testing.T) {
	mbm := &MagicBlockMap{Hash: "hash", BlockRound: 2}
	blob, err := json.Marshal(mbm)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	tests := []struct {
		name   string
		fields fields
		want   []byte
	}{
		{
			name: "OK",
			fields: fields{
				IDField:    mbm.IDField,
				Hash:       mbm.Hash,
				BlockRound: mbm.BlockRound,
			},
			want: blob,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if got := mb.Encode(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Encode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlockMap_Decode(t *testing.T) {
	mbm := &MagicBlockMap{Hash: "hash", BlockRound: 2}
	blob, err := json.Marshal(mbm)
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		IDField    datastore.IDField
		Hash       string
		BlockRound int64
	}
	type args struct {
		input []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		want    *MagicBlockMap
	}{
		{
			name: "OK",
			fields: fields{
				IDField:    mbm.IDField,
				Hash:       mbm.Hash,
				BlockRound: mbm.BlockRound,
			},
			args:    args{input: blob},
			want:    mbm,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mb := &MagicBlockMap{
				IDField:    tt.fields.IDField,
				Hash:       tt.fields.Hash,
				BlockRound: tt.fields.BlockRound,
			}
			if err := mb.Decode(tt.args.input); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(mb, tt.want) {
				t.Errorf("Decode() error = %v, want = %v", mb, tt.want)
			}
		})
	}
}
