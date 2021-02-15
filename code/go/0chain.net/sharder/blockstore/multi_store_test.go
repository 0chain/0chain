package blockstore

import (
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

func TestMultiBlockStore_DeleteBlock(t *testing.T) {
	t.Parallel()

	var (
		bs = makeTestFSBlockStore("tmp/test/multistore")

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)

	type fields struct {
		BlockStores []BlockStore
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_MultiBlockStore_DeleteBlock_OK",
			fields:  fields{BlockStores: []BlockStore{bs}},
			args:    args{b: &b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}

			if err := mbs.Write(tt.args.b); err != nil {
				t.Fatal(err)
			}

			if err := mbs.DeleteBlock(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("DeleteBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiBlockStore_Read(t *testing.T) {
	var (
		bs = makeTestFSBlockStore("tmp/test/multistore")

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)

	type fields struct {
		BlockStores []BlockStore
	}
	type args struct {
		hash  string
		round int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.Block
		wantErr bool
	}{
		{
			name:    "Test_MultiBlockStore_Read_OK",
			fields:  fields{BlockStores: []BlockStore{bs}},
			args:    args{hash: b.Hash},
			want:    &b,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}

			if err := mbs.Write(&b); err != nil {
				t.Fatal(err)
			}

			got, err := mbs.Read(tt.args.hash, tt.args.round)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !reflect.DeepEqual(got.Hash, tt.want.Hash) {
				t.Errorf("Read() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiBlockStore_Write(t *testing.T) {
	t.Parallel()

	var (
		bs = makeTestFSBlockStore("tmp/test/multistore")

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)

	type fields struct {
		BlockStores []BlockStore
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "TestMultiBlockStore_Write_OK",
			fields:  fields{BlockStores: []BlockStore{bs}},
			args:    args{b: &b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}
			if err := mbs.Write(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewMultiBlockStore(t *testing.T) {
	t.Parallel()

	bs := makeTestFSBlockStore("tmp/test/multistore")

	type args struct {
		blockstores []BlockStore
	}
	tests := []struct {
		name string
		args args
		want *MultiBlockStore
	}{
		{
			name: "TestNewMultiBlockStore_OK",
			args: args{
				blockstores: []BlockStore{
					bs,
					bs,
				},
			},
			want: &MultiBlockStore{
				[]BlockStore{
					bs,
					bs,
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewMultiBlockStore(tt.args.blockstores); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewMultiBlockStore() = %v, want %v", got, tt.want)
			}
		})
	}
}
