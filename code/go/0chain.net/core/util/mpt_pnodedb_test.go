package util

import (
	"context"
	"github.com/0chain/gorocksdb"
	"golang.org/x/exp/rand"
	"os"
	"reflect"
	"strconv"
	"sync"
	"testing"
)

const dataDir = "tmp"

func cleanUp() error {
	if err := os.RemoveAll(dataDir); err != nil {
		return err
	}

	return nil
}

func makeTestDB() (*gorocksdb.DB, error) {
	opts := gorocksdb.NewDefaultOptions()
	opts.SetCreateIfMissing(true)

	return gorocksdb.OpenDb(opts, dataDir)
}

func TestNewPNodeDB(t *testing.T) {
	sstType = SSTTypePlainTable

	db, err := makeTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	type args struct {
		dataDir string
		logDir  string
	}
	tests := []struct {
		name    string
		args    args
		want    *PNodeDB
		wantErr bool
	}{
		{
			name:    "Test_NewPNodeDB_ERR",
			args:    args{dataDir: dataDir},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewPNodeDB(tt.args.dataDir, tt.args.logDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewPNodeDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPNodeDB() got = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_Iterate(t *testing.T) {
	db, err := makeTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wo := gorocksdb.NewDefaultWriteOptions()
	db.Put(wo, []byte("key"), make([]byte, 0))

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	type args struct {
		ctx     context.Context
		handler NodeDBIteratorHandler
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_PNodeDB_Iterate_Err_Reading_OK",
			fields:  fields{db: db},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if err := pndb.Iterate(tt.args.ctx, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("Iterate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_PruneBelowVersion(t *testing.T) {
	db, err := makeTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wo := gorocksdb.NewDefaultWriteOptions()
	db.Put(wo, []byte("key"), []byte{NodeTypeValueNode})

	fo := gorocksdb.NewDefaultFlushOptions()
	ctx := context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{})

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	type args struct {
		ctx     context.Context
		version Sequence
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_PNodeDB_PruneBelowVersion_OK",
			fields:  fields{db: db, fo: fo},
			args:    args{ctx: ctx},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if err := pndb.PruneBelowVersion(tt.args.ctx, tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("PruneBelowVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_PruneBelowVersion_Iterator_Err(t *testing.T) {
	db, err := makeTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wo := gorocksdb.NewDefaultWriteOptions()

	for i := 0; i < 257; i++ {
		r := rand.Int()
		db.Put(wo, []byte(strconv.Itoa(r)), []byte{NodeTypeValueNode})
	}

	wo.DisableWAL(true)
	wo.SetSync(true)

	fo := gorocksdb.NewDefaultFlushOptions()
	ctx := context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{})

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	type args struct {
		ctx     context.Context
		version Sequence
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_PNodeDB_PruneBelowVersion_OK",
			fields:  fields{db: db, wo: wo, fo: fo},
			args:    args{ctx: ctx, version: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if err := pndb.PruneBelowVersion(tt.args.ctx, tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("PruneBelowVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_PruneBelowVersion_Multi_Delete_Err(t *testing.T) {
	db, err := makeTestDB()
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	wo := gorocksdb.NewDefaultWriteOptions()
	n := NewValueNode()
	db.Put(wo, []byte("key"), n.Encode())

	wo.DisableWAL(true)
	wo.SetSync(true)

	fo := gorocksdb.NewDefaultFlushOptions()
	ctx := context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{})

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	type args struct {
		ctx     context.Context
		version Sequence
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "Test_PNodeDB_PruneBelowVersion_OK",
			fields:  fields{db: db, wo: wo, fo: fo},
			args:    args{ctx: ctx, version: 1},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if err := pndb.PruneBelowVersion(tt.args.ctx, tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("PruneBelowVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_Close(t *testing.T) {
	db, err := makeTestDB()
	if err != nil {
		t.Fatal(err)
	}

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "Test_PNodeDB_Close_OK",
			fields: fields{db: db},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}

			pndb.Close()
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_GetDBVersions(t *testing.T) {
	v := []int64{1}

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	tests := []struct {
		name   string
		fields fields
		want   []int64
	}{
		{
			name:   "Test_PNodeDB_GetDBVersions_OK",
			fields: fields{versions: v},
			want:   v,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if got := pndb.GetDBVersions(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDBVersions() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Error(err)
	}
}

func TestPNodeDB_TrackDBVersion(t *testing.T) {
	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
		mutex    sync.Mutex
		version  int64
		versions []int64
	}
	type args struct {
		v int64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   []int64
	}{
		{
			name: "Test_PNodeDB_TrackDBVersion_OK",
			args: args{v: 1},
			want: []int64{1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				mutex:    tt.fields.mutex,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}

			pndb.TrackDBVersion(tt.args.v)

			if !reflect.DeepEqual(pndb.versions, tt.want) {
				t.Errorf("TrackDBVersion() got = %v, want = %v", pndb.versions, tt.want)
			}
		})
	}
}
