package blockstore

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/minio/minio-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/chain"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/core/viper"
	"github.com/0chain/common/core/logging"
)

func init() {
	serverChain := chain.NewChainFromConfig()
	conf := serverChain.ChainConfig.(*chain.ConfigImpl)
	conf.ConfDataForTest().RoundRange = 1
	chain.SetServerChain(serverChain)

	memoryStorage := memorystore.GetStorageProvider()
	block.SetupEntity(memoryStorage)

	block.SetupBlockSummaryEntity(ememorystore.GetStorageProvider())

	logging.InitLogging("testing", "")

	viper.Set("minio.enabled", true)
}

type (
	// implements MinioClient interface.
	minioClientMock struct{}
)

func (mock minioClientMock) FPutObject(_ string, hash string, _ string, _ minio.PutObjectOptions) (int64, error) {
	if len(hash) != 64 {
		return 0, errors.New("hash must be 64 size")
	}

	return 0, nil
}

func (mock minioClientMock) FGetObject(_ string, _ string, _ string, _ minio.GetObjectOptions) error {
	return nil
}

func (mock minioClientMock) StatObject(_ string, hash string, _ minio.StatObjectOptions) (minio.ObjectInfo, error) {
	if len(hash) != 64 {
		return minio.ObjectInfo{}, errors.New("hash must be 64 size")
	}
	return minio.ObjectInfo{}, nil
}

func (mock minioClientMock) MakeBucket(_ string, _ string) error {
	return nil
}

func (mock minioClientMock) BucketExists(_ string) (bool, error) {
	return false, nil
}

func (mock minioClientMock) BucketName() string {
	return ""
}

func (mock minioClientMock) DeleteLocal() bool {
	return true
}

func makeTestFSBlockStore(t *testing.T) (*FSBlockStore, func()) {
	tmpDir, err := os.MkdirTemp("", "store")
	require.NoError(t, err)

	bs := NewFSBlockStore(tmpDir, &minioClientMock{})
	cleanUp := func() {
		err := os.RemoveAll(filepath.Join(tmpDir))
		require.NoError(t, err)
	}

	return bs, cleanUp
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
		Minio                 MinioClient
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
	var (
		bs, cleanUp = makeTestFSBlockStore(t)

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)
	defer cleanUp()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
	}
	type args struct {
		b *block.Block
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		write   bool // writing file before starting read
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
			write:   true,
			wantErr: false,
		},
		{
			name: "Test_FSBlockStore_DeleteBlock_ERR",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args: func() args {
				b := block.NewBlock("", 1)
				b.Hash = encryption.Hash("another data")

				return args{b: b}
			}(),
			write:   false,
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

			if tt.write {
				if err := fbs.Write(tt.args.b); err != nil {
					t.Fatal(err)
				}
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
	var (
		bs, cleanUp = makeTestFSBlockStore(t)

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)
	defer cleanUp()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
			name: "Test_FSBlockStore_Read_From_File_OK",
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
	fbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
			want: filepath.Join(fbs.RootDirectory, "1/efd/a89/3aa/850b0c0e61f33325615b9d93bcf6b42d60d8f5d37ebc720fd4e3daf.dat.zlib"),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fbs := &FSBlockStore{
				RootDirectory:         tt.fields.RootDirectory,
				blockMetadataProvider: tt.fields.blockMetadataProvider,
				Minio:                 tt.fields.Minio,
			}
			got := fbs.getFileName(tt.args.hash, tt.args.round)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFSBlockStore_getFileWithoutExtension(t *testing.T) {
	fbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
			want: filepath.Join(fbs.RootDirectory, "1/efd/a89/3aa/850b0c0e61f33325615b9d93bcf6b42d60d8f5d37ebc720fd4e3daf"),
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
			got := fbs.getFileWithoutExtension(tt.args.hash, tt.args.round)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFSBlockStore_read(t *testing.T) {
	var (
		bs, cleanUp = makeTestFSBlockStore(t)

		b = block.Block{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)
	defer cleanUp()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
		write   bool // writing file before starting read
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
			write:   true,
			want:    &b,
			wantErr: false,
		},
		{
			name: "Test_FSBlockStore_read_ERR",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args: args{
				hash:  encryption.Hash("another data"),
				round: 1,
			},
			write:   false,
			wantErr: true,
		},
		{
			name: "Test_FSBlockStore_read_Invalid_Hash_Size_ERR",
			fields: fields{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			},
			args: args{
				hash:  b.Hash[:62],
				round: b.Round,
			},
			write:   false,
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

			if tt.write {
				if err := fbs.Write(&b); err != nil {
					t.Fatal(err)
				}
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

func TestFSBlockStore_UploadToCloud(t *testing.T) {
	t.Parallel()

	fsbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
			name: "Test_FSBlockStore_UploadToCloud_OK",
			fields: fields{
				RootDirectory:         fsbs.RootDirectory,
				blockMetadataProvider: fsbs.blockMetadataProvider,
				Minio:                 fsbs.Minio,
			},
			args: args{
				hash:  encryption.Hash("some data"),
				round: 1,
			},
			wantErr: false,
		},
		{
			name: "Test_FSBlockStore_UploadToCloud_ERR",
			fields: fields{
				RootDirectory:         fsbs.RootDirectory,
				blockMetadataProvider: fsbs.blockMetadataProvider,
				Minio:                 fsbs.Minio,
			},
			args: args{
				hash:  encryption.Hash("some data")[:62],
				round: 1,
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
			if err := fbs.UploadToCloud(tt.args.hash, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("UploadToCloud() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFSBlockStore_DownloadFromCloud(t *testing.T) {
	t.Parallel()

	fsbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
			name: "Test_FSBlockStore_DownloadFromCloud_OK",
			fields: fields{
				RootDirectory:         fsbs.RootDirectory,
				blockMetadataProvider: fsbs.blockMetadataProvider,
				Minio:                 fsbs.Minio,
			},
			args: args{
				hash:  b.Hash,
				round: b.Round,
			},
			wantErr: false,
		},
		{
			name: "Test_FSBlockStore_DownloadFromCloud_ERR",
			fields: fields{
				RootDirectory:         fsbs.RootDirectory,
				blockMetadataProvider: fsbs.blockMetadataProvider,
				Minio:                 fsbs.Minio,
			},
			args: args{
				hash:  b.Hash[:62],
				round: b.Round,
			},
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
			if err := fbs.DownloadFromCloud(tt.args.hash, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("DownloadFromCloud() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFSBlockStore_CloudObjectExists(t *testing.T) {
	t.Parallel()

	fsbs, cleanUp := makeTestFSBlockStore(t)
	defer cleanUp()
	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

	type fields struct {
		RootDirectory         string
		blockMetadataProvider datastore.EntityMetadata
		Minio                 MinioClient
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
			name: "Test_FSBlockStore_DownloadFromCloud_TRUE",
			fields: fields{
				RootDirectory:         fsbs.RootDirectory,
				blockMetadataProvider: fsbs.blockMetadataProvider,
				Minio:                 fsbs.Minio,
			},
			args: args{b.Hash},
			want: true,
		},
		{

			name: "Test_FSBlockStore_DownloadFromCloud_FALSE",
			fields: fields{
				RootDirectory:         fsbs.RootDirectory,
				blockMetadataProvider: fsbs.blockMetadataProvider,
				Minio:                 fsbs.Minio,
			},
			args: args{hash: b.Hash[:62]},
			want: false,
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
			if got := fbs.CloudObjectExists(tt.args.hash); got != tt.want {
				t.Errorf("CloudObjectExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFSBlockStore_ReadWithBlockSummary(t *testing.T) {
	t.Parallel()

	b := block.Block{
		HashIDField: datastore.HashIDField{
			Hash: encryption.Hash("bs data"),
		},
		MagicBlock: &block.MagicBlock{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("mb data"),
			},
			T: 1,
		},
	}
	b.Round = 0

	type args struct {
		bs *block.BlockSummary
	}
	tests := []struct {
		name    string
		args    args
		want    *block.Block
		wantErr bool
	}{
		{
			name:    "Test_FSBlockStore_ReadWithBlockSummary_OK",
			want:    &b,
			args:    args{b.GetSummary()},
			wantErr: false,
		},
		{
			name:    "Test_FSBlockStore_ReadWithBlockSummary_ERR",
			want:    &b,
			args:    args{b.GetSummary()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bs, cleanUp := makeTestFSBlockStore(t)
			defer cleanUp()

			fbs := &FSBlockStore{
				RootDirectory:         bs.RootDirectory,
				blockMetadataProvider: bs.blockMetadataProvider,
				Minio:                 bs.Minio,
			}

			if !tt.wantErr {
				err := fbs.Write(&b)
				require.NoError(t, err, err)
				return
			}

			got, err := fbs.ReadWithBlockSummary(tt.args.bs)
			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.Equal(t, tt.want, got)
		})
	}
}
