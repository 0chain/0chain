package block

import (
	"context"
	"reflect"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	"0chain.net/core/datastore"
	"0chain.net/core/mocks"
)

func init() {
	SetupMagicBlockData(&mocks.Store{})
}

func TestMagicBlockData_GetEntityMetadata(t *testing.T) {
	type fields struct {
		IDField    datastore.IDField
		MagicBlock *MagicBlock
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.EntityMetadata
	}{
		{
			name: "OK",
			want: magicBlockMetadata,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &MagicBlockData{
				IDField:    tt.fields.IDField,
				MagicBlock: tt.fields.MagicBlock,
			}
			if got := m.GetEntityMetadata(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityMetadata() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMagicBlockDataProvider(t *testing.T) {
	tests := []struct {
		name string
		want datastore.Entity
	}{
		{
			name: "OK",
			want: &MagicBlockData{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := MagicBlockDataProvider(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MagicBlockDataProvider() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSetupMagicBlockDataDB_Panic(t *testing.T) {
	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "OK",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("SetupBlockSummaryDB() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			SetupMagicBlockDataDB("")
		})
	}
}

func TestMagicBlockData_Read(t *testing.T) {
	store := mocks.Store{}
	store.On("Read", context.Context(nil), "", new(MagicBlockData)).Return(
		func(_ context.Context, _ string, _ datastore.Entity) error {
			return nil
		},
	)

	SetupMagicBlockData(&store)

	type fields struct {
		IDField    datastore.IDField
		MagicBlock *MagicBlock
	}
	type args struct {
		ctx context.Context
		key string
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
			m := &MagicBlockData{
				IDField:    tt.fields.IDField,
				MagicBlock: tt.fields.MagicBlock,
			}
			if err := m.Read(tt.args.ctx, tt.args.key); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMagicBlockData_Write(t *testing.T) {
	store := mocks.Store{}
	store.On("Write", context.Context(nil), new(MagicBlockData)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	SetupMagicBlockData(&store)

	type fields struct {
		IDField    datastore.IDField
		MagicBlock *MagicBlock
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
			m := &MagicBlockData{
				IDField:    tt.fields.IDField,
				MagicBlock: tt.fields.MagicBlock,
			}
			if err := m.Write(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMagicBlockData_Delete(t *testing.T) {
	store := mocks.Store{}
	store.On("Delete", context.Context(nil), new(MagicBlockData)).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)

	SetupMagicBlockData(&store)

	type fields struct {
		IDField    datastore.IDField
		MagicBlock *MagicBlock
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
			m := &MagicBlockData{
				IDField:    tt.fields.IDField,
				MagicBlock: tt.fields.MagicBlock,
			}
			if err := m.Delete(tt.args.ctx); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewMagicBlockData(t *testing.T) {
	mb := NewMagicBlock()
	mb.MagicBlockNumber = 2

	mbData, ok := datastore.GetEntityMetadata("magicblockdata").Instance().(*MagicBlockData)
	require.True(t, ok)

	mbData.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)
	mbData.MagicBlock = mb

	type args struct {
		mb *MagicBlock
	}
	tests := []struct {
		name string
		args args
		want *MagicBlockData
	}{
		{
			name: "OK",
			args: args{mb: mb},
			want: mbData,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMagicBlockData(tt.args.mb); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMagicBlockData() = %v, want %v", got, tt.want)
			}
		})
	}
}
