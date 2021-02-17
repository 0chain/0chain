package blockstore

import (
	"os"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
)

func init() {
	serverChain := chain.NewChainFromConfig()
	serverChain.RoundRange = 1
	chain.SetServerChain(serverChain)

	memoryStorage := memorystore.GetStorageProvider()
	block.SetupEntity(memoryStorage)
}

func makeTestFSBlockStore(dir string) *FSBlockStore {
	bs := NewFSBlockStore(dir, &MinioClient{})
	return bs
}

// checkFile returns true if file exist.
func checkFile(fileName string) bool {
	f, err := os.Open(fileName)
	if err != nil {
		return false
	}
	_ = f.Close()

	return true
}

func TestFSBlockStore_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 *MinioClient
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
			name:    "Test_FSBlockStore_Delete_ERR",
			wantErr: true, // want err because FSBlockStore does not provide this interface
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}
			if err := fbs.Delete(tt.args.hash); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFSBlockStore_DeleteBlock(t *testing.T) {
	t.Parallel()

	var (
		bs = makeTestFSBlockStore("tmp/test/fsblockstore")

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 *MinioClient
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
			name: "Test_FSBlockStore_DeleteBlock_OK",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args:    args{b: &b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}

			if err := fbs.Write(tt.args.b); err != nil {
				t.Fatal(err)
			}

			if err := fbs.DeleteBlock(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("DeleteBlock() error = %v, wantErr %v", err, tt.wantErr)
			}

			saved := checkFile(fbs.getFileName(tt.args.b.Hash, tt.args.b.Round))
			if !tt.wantErr && saved {
				t.Errorf("DeleteBlock() saved = %v, wantErr %v", saved, tt.wantErr)
			}
		})
	}
}

func TestFSBlockStore_Read(t *testing.T) {
	t.Parallel()

	var (
		bs = makeTestFSBlockStore("tmp/test/fsblockstore")

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)
	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 *MinioClient
	}
	type args struct {
		hash  string
		round int64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantB   *block.Block
		wantErr bool
	}{
		{
			name: "Test_FSBlockStore_Read_OK",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args: args{
				hash:  b.Hash,
				round: b.Round,
			},
			wantB:   &b,
			wantErr: false,
		},
		{
			name: "Test_FSBlockStore_Read_Invalid_Hash_Length_OK",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args: args{
				hash:  b.Hash[:63],
				round: b.Round,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}

			if err := fbs.Write(&b); err != nil {
				t.Fatal(err)
			}

			gotB, err := fbs.Read(tt.args.hash, tt.args.round)
			if (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotB != nil && !reflect.DeepEqual(gotB.Hash, tt.wantB.Hash) {
				t.Errorf("Read() gotB = %#v, want %#v", gotB, tt.wantB)
			}
		})
	}
}

func TestFSBlockStore_getFileName(t *testing.T) {
	t.Parallel()

	var (
		fbs = makeTestFSBlockStore("tmp/test/fsblockstore")
	)

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 *MinioClient
	}
	type args struct {
		hash  string
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "TestFSBlockStore_getFileName_OK",
			fields: fields{
				RootDirectory:         fbs.RootDirectory,
				blockMetadataProvider: fbs.blockMetadataProvider,
				Minio:                 fbs.Minio,
			},
			args: args{
				hash:  encryption.Hash("data"),
				round: 1,
			},
			want: "tmp/test/fsblockstore/1/efd/a89/3aa/850b0c0e61f33325615b9d93bcf6b42d60d8f5d37ebc720fd4e3daf.dat.zlib",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}
			if got := fbs.getFileName(tt.args.hash, tt.args.round); got != tt.want {
				t.Errorf("getFileName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFSBlockStore_getFileWithoutExtension(t *testing.T) {
	t.Parallel()

	var (
		fbs = makeTestFSBlockStore("tmp/test/fsblockstore")
	)
	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 *MinioClient
	}
	type args struct {
		hash  string
		round int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{
		{
			name: "Test_FSBlockStore_getFileNameWithoutExtension_OK",
			fields: fields{
				RootDirectory:         fbs.RootDirectory,
				blockMetadataProvider: fbs.blockMetadataProvider,
				Minio:                 fbs.Minio,
			},
			args: args{
				hash:  encryption.Hash("data"),
				round: 1,
			},
			want: "tmp/test/fsblockstore/1/efd/a89/3aa/850b0c0e61f33325615b9d93bcf6b42d60d8f5d37ebc720fd4e3daf",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}
			if got := fbs.getFileWithoutExtension(tt.args.hash, tt.args.round); got != tt.want {
				t.Errorf("getFileWithoutExtension() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFSBlockStore_read(t *testing.T) {
	t.Parallel()

	var (
		bs = makeTestFSBlockStore("tmp/test/fsblockstore")

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)
	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 *MinioClient
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
			name: "Test_FSBlockStore_read_OK",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args: args{
				hash:  b.Hash,
				round: b.Round,
			},
			want:    &b,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}

			if err := fbs.Write(&b); err != nil {
				t.Fatal(err)
			}

			got, err := fbs.read(tt.args.hash, tt.args.round)
			if (err != nil) != tt.wantErr {
				t.Errorf("read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != nil && !reflect.DeepEqual(got.Hash, tt.want.Hash) {
				t.Errorf("read() got = %v, want %v", got, tt.want)
			}
		})
	}
}
