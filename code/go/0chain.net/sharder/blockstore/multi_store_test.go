package blockstore

import (
	"errors"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
)

type blockStoreMock struct{}

func (b2 blockStoreMock) Write(b *block.Block) error {
	if len(b.Hash) != 64 {
		return errors.New("hash must be 64 size")
	}
	return nil
}

func (b2 blockStoreMock) Read(_ string, _ int64) (*block.Block, error) {
	return nil, nil
}

func (b2 blockStoreMock) ReadWithBlockSummary(_ *block.BlockSummary) (*block.Block, error) {
	return nil, nil
}

func (b2 blockStoreMock) Delete(hash string) error {
	if len(hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	return nil
}

func (b2 blockStoreMock) DeleteBlock(b *block.Block) error {
	if len(b.Hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	return nil
}

func (b2 blockStoreMock) UploadToCloud(_ string, _ int64) error {
	return nil
}

func (b2 blockStoreMock) DownloadFromCloud(_ string, _ int64) error {
	return nil
}

func (b2 blockStoreMock) CloudObjectExists(_ string) bool {
	return false
}

var _ BlockStore = (*blockStoreMock)(nil)

func TestMultiBlockStore_DeleteBlock(t *testing.T) {
	t.Parallel()

	var (
		bs = blockStoreMock{}

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
		{
			name:   "Test_MultiBlockStore_DeleteBlock_ERR",
			fields: fields{BlockStores: []BlockStore{bs}},
			args: func() args {
				b := block.Block{
					HashIDField: datastore.HashIDField{
						Hash: encryption.Hash("data"),
					},
				}
				b.Hash = b.Hash[:62]

				return args{b: &b}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}

			if err := mbs.DeleteBlock(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("DeleteBlock() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiBlockStore_Read(t *testing.T) {
	var (
		bs = blockStoreMock{}

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
		bs = blockStoreMock{}

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
			name:    "Test_MultiBlockStore_Write_OK",
			fields:  fields{BlockStores: []BlockStore{bs}},
			args:    args{b: &b},
			wantErr: false,
		},
		{
			name:   "Test_MultiBlockStore_Write_ERR",
			fields: fields{BlockStores: []BlockStore{bs}},
			args: func() args {
				b := block.Block{
					HashIDField: datastore.HashIDField{
						Hash: encryption.Hash("data"),
					},
				}
				b.Hash = b.Hash[:62]

				return args{b: &b}
			}(),
			wantErr: true,
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

	bs := blockStoreMock{}

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

func TestMultiBlockStore_CloudObjectExists(t *testing.T) {
	t.Parallel()

	bs := blockStoreMock{}

	type fields struct {
		BlockStores []BlockStore
	}
	type args struct {
		hash string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "Test_MultiBlockStore_CloudObjectExists_OK",
			fields: fields{
				[]BlockStore{
					bs,
					bs,
				},
			},
			want: false, // because method is not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}
			if got := mbs.CloudObjectExists(tt.args.hash); got != tt.want {
				t.Errorf("CloudObjectExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiBlockStore_DownloadFromCloud(t *testing.T) {
	t.Parallel()

	bs := blockStoreMock{}

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
		wantErr bool
	}{
		{
			name: "Test_MultiBlockStore_DownloadFromCloud_OK",
			fields: fields{
				[]BlockStore{
					bs,
					bs,
				},
			},
			wantErr: true, // because method is not implemented
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}
			if err := mbs.DownloadFromCloud(tt.args.hash, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("DownloadFromCloud() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiBlockStore_UploadToCloud(t *testing.T) {
	t.Parallel()

	bs := blockStoreMock{}

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
		wantErr bool
	}{
		{
			name: "Test_MultiBlockStore_UploadToCloud_OK",
			fields: fields{
				[]BlockStore{
					bs,
					bs,
				},
			},
			wantErr: true, // because method is not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}
			if err := mbs.UploadToCloud(tt.args.hash, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("UploadToCloud() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMultiBlockStore_ReadWithBlockSummary(t *testing.T) {
	t.Parallel()

	bs := blockStoreMock{}

	type fields struct {
		BlockStores []BlockStore
	}
	type args struct {
		bs *block.BlockSummary
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *block.Block
		wantErr bool
	}{
		{
			name: "Test_MultiBlockStore_ReadWithBlockSummary_OK",
			fields: fields{
				[]BlockStore{
					bs,
					bs,
				},
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}
			got, err := mbs.ReadWithBlockSummary(tt.args.bs)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadWithBlockSummary() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadWithBlockSummary() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMultiBlockStore_Delete(t *testing.T) {
	t.Parallel()

	bs := blockStoreMock{}

	type fields struct {
		BlockStores []BlockStore
	}
	type args struct {
		hash string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Test_MultiBlockStore_ReadWithBlockSummary_OK",
			fields: fields{
				[]BlockStore{
					bs,
					bs,
				},
			},
			args:    args{hash: encryption.Hash("data")},
			wantErr: false,
		},
		{
			name:   "Test_MultiBlockStore_Write_ERR",
			fields: fields{BlockStores: []BlockStore{bs}},
			args: func() args {
				h := encryption.Hash("data")
				h = h[:62]

				return args{hash: h}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mbs := &MultiBlockStore{
				BlockStores: tt.fields.BlockStores,
			}
			if err := mbs.Delete(tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
