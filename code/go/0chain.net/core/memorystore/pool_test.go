package memorystore

import (
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func init() {
	logging.InitLogging("development")
}

func initDefaultPool() error {
	mr, err := miniredis.Run()
	if err != nil {
		return err
	}

	DefaultPool = &redis.Pool{
		MaxIdle:   80,
		MaxActive: 1000, // max number of connections
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", mr.Addr())
			if err != nil {
				panic(err.Error())
			}
			return c, err
		},
	}

	return nil
}

func TestNewPool(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatal(err)
	}
	portPos := strings.Index(mr.Addr(), ":")
	portInt, err := strconv.Atoi(mr.Addr()[portPos+1:])
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		host string
		port int
	}
	tests := []struct {
		name          string
		args          args
		want          *redis.Pool
		wantPanic     bool
		wantDialCheck bool
	}{
		{
			name: "Test_NewPool_OK",
			args: args{port: 8080},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000,
			},
		},
		{
			name: "Test_NewPool_Panic",
			args: args{port: 8080},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000,
			},
			wantPanic:     true,
			wantDialCheck: true,
		},
		{
			name: "Test_NewPool_Dial_Check_OK",
			args: args{port: portInt},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000,
			},
			wantDialCheck: true,
		},
		{
			name: "Test_NewPool_OK",
			args: args{port: 8080},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000,
			},
		},
		{
			name: "Test_NewPool_Panic",
			args: args{port: 8080},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000,
			},
			wantPanic:     true,
			wantDialCheck: true,
		},
		{
			name: "Test_NewPool_Dial_Check_OK",
			args: args{port: portInt},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000,
			},
			wantDialCheck: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("getConnectionCount() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got := NewPool(tt.args.host, tt.args.port)
			if tt.wantDialCheck {
				if _, err := got.Dial(); err != nil {
					t.Fatal(err)
				}
			}
			got.Dial = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPool() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestNewPool_Docker(t *testing.T) {
	if err := os.Setenv("DOCKER", "docker"); err != nil {
		t.Fatal(err)
	}

	type args struct {
		host string
		port int
	}
	tests := []struct {
		name string
		args args
		want *redis.Pool
	}{
		{
			name: "Test_NewPool_OK",
			args: args{host: "host"},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000, // max number of connections
			},
		},
		{
			name: "Test_NewPool_OK",
			args: args{host: "host"},
			want: &redis.Pool{
				MaxIdle:   80,
				MaxActive: 1000, // max number of connections
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := NewPool(tt.args.host, tt.args.port)
			got.Dial = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPool() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAddPool(t *testing.T) {
	const dbid = "dbid"
	pool := NewPool("", 8080)

	type args struct {
		dbid string
		pool *redis.Pool
	}
	tests := []struct {
		name string
		args args
		want map[string]*dbpool
	}{
		{
			name: "Test_AddPool_OK",
			args: args{dbid: dbid, pool: pool},
			want: func() map[string]*dbpool {
				p := make(map[string]*dbpool)
				for key, value := range pools.list {
					p[key] = value
				}

				p[dbid] = &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}
				return p
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			AddPool(tt.args.dbid, tt.args.pool)
			require.EqualValues(t, tt.want[tt.args.dbid], pools.list[tt.args.dbid])
		})
	}
}

func TestGetConnectionCount(t *testing.T) {
	dbid := "dbid"
	pool := NewPool("", 8080)
	pools.list[dbid] = &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}

	type args struct {
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name      string
		args      args
		want      int
		want1     int
		wantPanic bool
	}{
		{
			name:  "Test_GetConnectionCount_OK",
			args:  args{entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want:  pool.ActiveCount(),
			want1: pool.IdleCount(),
		},
		{
			name:      "Test_GetConnectionCount_Panic",
			args:      args{entityMetadata: &datastore.EntityMetadataImpl{DB: "unknown"}},
			wantPanic: true,
		},
		{
			name:  "Test_GetConnectionCount_OK",
			args:  args{entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want:  pool.ActiveCount(),
			want1: pool.IdleCount(),
		},
		{
			name:      "Test_GetConnectionCount_Panic",
			args:      args{entityMetadata: &datastore.EntityMetadataImpl{DB: "unknown"}},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("getConnectionCount() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got, got1 := getConnectionCount(tt.args.entityMetadata)
			if got != tt.want {
				t.Errorf("getConnectionCount() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("getConnectionCount() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getdbpool(t *testing.T) {
	dbid := "dbid"
	pool := NewPool("", 8080)
	pools.list[dbid] = &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}

	type args struct {
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name      string
		args      args
		want      *dbpool
		wantPanic bool
	}{
		{
			name: "Test_getdbpool_OK",
			args: args{entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want: &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool},
		},
		{
			name:      "Test_getdbpool_Panic",
			args:      args{entityMetadata: &datastore.EntityMetadataImpl{DB: "unknown"}},
			wantPanic: true,
		},
		{
			name: "Test_getdbpool_OK",
			args: args{entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want: &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool},
		},
		{
			name:      "Test_getdbpool_Panic",
			args:      args{entityMetadata: &datastore.EntityMetadataImpl{DB: "unknown"}},
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("getDbPool() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := getDbPool(tt.args.entityMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getDbPool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInitDefaultPool(t *testing.T) {
	t.Parallel()

	type args struct {
		host string
		port int
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_InitDefaultPool_OK",
		},
		{
			name: "Test_InitDefaultPool_OK",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			InitDefaultPool(tt.args.host, tt.args.port)
		})
	}
}
