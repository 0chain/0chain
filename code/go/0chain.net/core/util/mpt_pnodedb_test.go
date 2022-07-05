package util

import (
	"context"
	"math/rand"
	"reflect"
	"strconv"
	"testing"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/require"
)

func TestPNodeDB_Iterate(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx     context.Context
		handler NodeDBIteratorHandler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test_PNodeDB_Iterate_Err_Reading_OK",
			args: args{
				handler: func(_ context.Context, _ Key, _ Node) error {
					return nil
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pndb, cleanUp := newPNodeDB(t)
			defer cleanUp()

			wo := gorocksdb.NewDefaultWriteOptions()
			for i := 0; i < 257; i++ {
				r := rand.Int()
				err := pndb.db.Put(wo, []byte(strconv.Itoa(r)), []byte{NodeTypeValueNode})
				require.NoError(t, err)
			}
			err := pndb.db.Put(wo, []byte("key"), make([]byte, 0))
			require.NoError(t, err)

			if err := pndb.Iterate(tt.args.ctx, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("Iterate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPNodeDB_PruneBelowVersion_Iterator_Err(t *testing.T) {
	t.Parallel()

	type fields struct {
		dataDir  string
		ro       *gorocksdb.ReadOptions
		to       *gorocksdb.TransactionOptions
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
			name: "Test_PNodeDB_PruneBelowVersion_OK",
			args: args{
				ctx:     context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{}),
				version: 1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, cleanUp := newPNodeDB(t)
			defer cleanUp()
			wo := gorocksdb.NewDefaultWriteOptions()
			for i := 0; i < 257; i++ {
				r := rand.Int()
				err := db.db.Put(wo, []byte(strconv.Itoa(r)), []byte{NodeTypeValueNode})
				require.NoError(t, err)
			}
			wo.DisableWAL(true)
			wo.SetSync(true)
			fo := gorocksdb.NewDefaultFlushOptions()
			ctx := context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{})
			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       db.db,
				ro:       tt.fields.ro,
				wo:       wo,
				to:       tt.fields.to,
				fo:       fo,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if err := pndb.PruneBelowVersion(ctx, tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("PruneBelowVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPNodeDB_PruneBelowVersion_Multi_Delete_Err(t *testing.T) {
	t.Parallel()

	type fields struct {
		dataDir  string
		ro       *gorocksdb.ReadOptions
		to       *gorocksdb.TransactionOptions
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
			name: "Test_PNodeDB_PruneBelowVersion_OK",
			args: args{
				ctx:     context.WithValue(context.TODO(), PruneStatsKey, &PruneStats{}),
				version: 1,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			db, cleanUp := newPNodeDB(t)
			defer cleanUp()

			wo := gorocksdb.NewDefaultWriteOptions()
			for i := 0; i < 257; i++ {
				r := rand.Int()
				err := db.db.Put(wo, []byte(strconv.Itoa(r)), []byte{NodeTypeValueNode})
				require.NoError(t, err)
			}

			n := NewValueNode()
			err := db.db.Put(wo, []byte("key"), n.Encode())
			require.NoError(t, err)

			wo.DisableWAL(true)
			wo.SetSync(true)

			fo := gorocksdb.NewDefaultFlushOptions()

			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       db.db,
				ro:       tt.fields.ro,
				wo:       wo,
				to:       tt.fields.to,
				fo:       fo,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if err := pndb.PruneBelowVersion(tt.args.ctx, tt.args.version); (err != nil) != tt.wantErr {
				t.Errorf("PruneBelowVersion() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPNodeDB_GetDBVersions(t *testing.T) {
	t.Parallel()

	v := []int64{1}

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
				version:  tt.fields.version,
				versions: tt.fields.versions,
			}
			if got := pndb.GetDBVersions(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetDBVersions() = %v, want %v", got, tt.want)
			}
		})
	}

}

func TestPNodeDB_TrackDBVersion(t *testing.T) {
	t.Parallel()

	type fields struct {
		dataDir  string
		db       *gorocksdb.DB
		ro       *gorocksdb.ReadOptions
		wo       *gorocksdb.WriteOptions
		to       *gorocksdb.TransactionOptions
		fo       *gorocksdb.FlushOptions
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			pndb := &PNodeDB{
				dataDir:  tt.fields.dataDir,
				db:       tt.fields.db,
				ro:       tt.fields.ro,
				wo:       tt.fields.wo,
				to:       tt.fields.to,
				fo:       tt.fields.fo,
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
