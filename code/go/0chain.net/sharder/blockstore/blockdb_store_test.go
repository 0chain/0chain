package blockstore

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"0chain.net/sharder/blockdb"
)

func makeTestBlockDBStore() *BlockDBStore {
	return &BlockDBStore{
		FSBlockStore:        makeTestFSBlockStore("tmp/test/blockdbstore"),
		txnMetadataProvider: datastore.GetEntityMetadata("txn"),
		compress:            true,
	}
}

func makeTestBlock() *block.Block {
	memoryStorage := memorystore.GetStorageProvider()
	block.SetupEntity(memoryStorage)

	b := block.NewBlock("", 1)
	b.Hash = encryption.Hash("data")

	return b
}

func TestNewBlockDBStore(t *testing.T) {
	t.Parallel()

	var (
		store = makeTestBlockDBStore()
	)

	type args struct {
		rootDir string
	}
	tests := []struct {
		name string
		args args
		want BlockStore
	}{
		{
			name: "Test_NewBlockDBStore_OK",
			args: args{rootDir: store.RootDirectory},
			want: store,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := NewBlockDBStore(tt.args.rootDir); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewBlockDBStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_blockHeader_Encode(t *testing.T) {
	t.Parallel()

	var (
		b = makeTestBlock()
	)

	type fields struct {
		Block *block.Block
	}
	tests := []struct {
		name       string
		fields     fields
		wantWriter string
		wantErr    bool
	}{
		{
			name: "Test_blockHeader_Encode_OK",
			fields: fields{
				Block: b,
			},
			wantWriter: func() string {
				buffer := bytes.NewBuffer(make([]byte, 0, 256))
				if _, err := datastore.ToMsgpack(b).WriteTo(buffer); err != nil {
					t.Fatal(err)
				}

				return buffer.String()
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bh := &blockHeader{
				Block: tt.fields.Block,
			}
			writer := &bytes.Buffer{}
			err := bh.Encode(writer)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWriter := writer.String(); gotWriter != tt.wantWriter {
				t.Errorf("Encode() gotWriter = %v, want %v", gotWriter, tt.wantWriter)
			}
		})
	}
}

func Test_blockHeader_Decode(t *testing.T) {
	t.Parallel()

	var (
		b = makeTestBlock()
	)

	type fields struct {
		Block *block.Block
	}
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    blockHeader
		wantErr bool
	}{
		{
			name:   "Test_blockHeader_Decode_OK",
			fields: fields{Block: &block.Block{}},
			args: func() args {
				var (
					buff = bytes.Buffer{}
					bh   = &blockHeader{
						Block: b,
					}
				)
				if err := bh.Encode(&buff); err != nil {
					t.Fatal(err)
				}

				return args{reader: &buff}
			}(),
			want:    blockHeader{Block: b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bh := blockHeader{
				Block: tt.fields.Block,
			}
			if err := bh.Decode(tt.args.reader); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_txnRecord_GetKey(t *testing.T) {
	t.Parallel()

	var (
		tx = &transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)

	type fields struct {
		Transaction *transaction.Transaction
	}
	tests := []struct {
		name   string
		fields fields
		want   blockdb.Key
	}{
		{
			name:   "Test_txnRecord_GetKey_OK",
			fields: fields{Transaction: tx},
			want:   blockdb.Key(tx.Hash),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &txnRecord{
				Transaction: tt.fields.Transaction,
			}
			if got := tr.GetKey(); got != tt.want {
				t.Errorf("GetKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_txnRecord_Encode(t *testing.T) {
	t.Parallel()

	var (
		tx = &transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
		}
	)

	type fields struct {
		Transaction *transaction.Transaction
	}
	tests := []struct {
		name       string
		fields     fields
		wantWriter string
		wantErr    bool
	}{
		{
			name: "Test_txnRecord_Encode_OK",
			fields: fields{
				Transaction: tx,
			},
			wantWriter: func() string {
				buffer := bytes.NewBuffer(make([]byte, 0, 256))
				if _, err := datastore.ToMsgpack(tx).WriteTo(buffer); err != nil {
					t.Fatal(err)
				}

				return buffer.String()
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &txnRecord{
				Transaction: tt.fields.Transaction,
			}
			writer := &bytes.Buffer{}
			err := tr.Encode(writer)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encode() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotWriter := writer.String(); gotWriter != tt.wantWriter {
				t.Errorf("Encode() gotWriter = %v, want %v", gotWriter, tt.wantWriter)
			}
		})
	}
}

func Test_txnRecord_Decode(t *testing.T) {
	t.Parallel()

	var (
		tx = &transaction.Transaction{
			HashIDField: datastore.HashIDField{
				Hash: encryption.Hash("data"),
			},
			ClientID: "1234",
		}
	)

	type fields struct {
		Transaction *transaction.Transaction
	}
	type args struct {
		reader io.Reader
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    txnRecord
		wantErr bool
	}{
		{
			name:   "Test_txnRecord_Decode_OK",
			fields: fields{Transaction: &transaction.Transaction{}},
			args: func() args {
				var (
					buff = bytes.Buffer{}
					bh   = &txnRecord{
						Transaction: tx,
					}
				)
				if err := bh.Encode(&buff); err != nil {
					t.Fatal(err)
				}

				return args{reader: &buff}
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tr := &txnRecord{
				Transaction: tt.fields.Transaction,
			}
			if err := tr.Decode(tt.args.reader); (err != nil) != tt.wantErr {
				t.Errorf("Decode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockDBStore_DeleteBlock(t *testing.T) {
	t.Parallel()

	var (
		db = makeTestBlockDBStore()
		b  = makeTestBlock()
	)

	type fields struct {
		FSBlockStore        *FSBlockStore
		txnMetadataProvider datastore.EntityMetadata
		compress            bool
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
			name: "Test_BlockDBStore_DeleteBlock_OK",
			fields: fields{
				FSBlockStore:        db.FSBlockStore,
				txnMetadataProvider: db.txnMetadataProvider,
				compress:            db.compress,
			},
			args: args{b: b},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bdbs := &BlockDBStore{
				FSBlockStore:        tt.fields.FSBlockStore,
				txnMetadataProvider: tt.fields.txnMetadataProvider,
				compress:            tt.fields.compress,
			}

			if err := bdbs.Write(tt.args.b); (err != nil) != tt.wantErr {
				t.Fatal(err)
			}

			if err := bdbs.DeleteBlock(tt.args.b); (err != nil) != tt.wantErr {
				t.Errorf("DeleteBlock() error = %v, wantErr %v", err, tt.wantErr)
			}

			file := bdbs.FSBlockStore.getFileWithoutExtension(tt.args.b.Hash, tt.args.b.Round)
			ext := blockdb.FileExtData
			saved := checkFile(file + "." + ext)
			if saved {
				t.Errorf("DeleteBlock() saved = %v, want %v", saved, false)
			}
		})
	}
}

func TestBlockDBStore_UploadToCloud(t *testing.T) {
	t.Parallel()

	type fields struct {
		FSBlockStore        *FSBlockStore
		txnMetadataProvider datastore.EntityMetadata
		compress            bool
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
			name:    "Test_BlockDBStore_UploadToCloud_ERR",
			wantErr: true, // want err because method not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bdbs := &BlockDBStore{
				FSBlockStore:        tt.fields.FSBlockStore,
				txnMetadataProvider: tt.fields.txnMetadataProvider,
				compress:            tt.fields.compress,
			}
			if err := bdbs.UploadToCloud(tt.args.hash, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("UploadToCloud() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockDBStore_DownloadFromCloud(t *testing.T) {
	t.Parallel()

	type fields struct {
		FSBlockStore        *FSBlockStore
		txnMetadataProvider datastore.EntityMetadata
		compress            bool
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
			name:    "Test_BlockDBStore_DownloadFromCloud_ERR",
			wantErr: true, // want err because method not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bdbs := &BlockDBStore{
				FSBlockStore:        tt.fields.FSBlockStore,
				txnMetadataProvider: tt.fields.txnMetadataProvider,
				compress:            tt.fields.compress,
			}
			if err := bdbs.DownloadFromCloud(tt.args.hash, tt.args.round); (err != nil) != tt.wantErr {
				t.Errorf("DownloadFromCloud() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBlockDBStore_CloudObjectExists(t *testing.T) {
	t.Parallel()

	type fields struct {
		FSBlockStore        *FSBlockStore
		txnMetadataProvider datastore.EntityMetadata
		compress            bool
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
			name: "TestBlockDBStore_CloudObjectExists_OK",
			want: false, // method not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			bdbs := &BlockDBStore{
				FSBlockStore:        tt.fields.FSBlockStore,
				txnMetadataProvider: tt.fields.txnMetadataProvider,
				compress:            tt.fields.compress,
			}
			if got := bdbs.CloudObjectExists(tt.args.hash); got != tt.want {
				t.Errorf("CloudObjectExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
