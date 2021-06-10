package block

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/mocks"
)

func init() {
	SetupHandlers()
}

func TestGetBlock(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		r   *http.Request
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "TRUE",
			args:    args{ctx: context.TODO(), r: httptest.NewRequest(http.MethodGet, "/", nil)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := GetBlock(tt.args.ctx, tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestPutBlock(t *testing.T) {
	t.Parallel()

	nwb := NewBlock("", 1)
	nwb.CreationDate = 0

	b := NewBlock("", 1)
	b.CreationDate = common.Now()
	store := mocks.Store{}
	store.On("Write", context.Context(nil), b).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return nil
		},
	)
	store.On("Write", context.TODO(), b).Return(
		func(_ context.Context, _ datastore.Entity) error {
			return errors.New("")
		},
	)

	blockEntityMetadata = &datastore.EntityMetadataImpl{
		Store: &store,
	}

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    interface{}
		wantErr bool
	}{
		{
			name:    "OK",
			args:    args{entity: b},
			want:    b,
			wantErr: false,
		},
		{
			name:    "Not_A_Block_ERR",
			args:    args{entity: new(BlockSummary)},
			wantErr: true,
		},
		{
			name:    "Not_Within_Tolerance_ERR",
			args:    args{entity: nwb},
			wantErr: true,
		},
		{
			name:    "Write_ERR",
			args:    args{ctx: context.TODO(), entity: b},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := PutBlock(tt.args.ctx, tt.args.entity)
			if (err != nil) != tt.wantErr {
				t.Errorf("PutBlock() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PutBlock() got = %v, want %v", got, tt.want)
			}
		})
	}
}
