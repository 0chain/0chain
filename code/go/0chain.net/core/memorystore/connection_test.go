package memorystore

import (
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/logging"
	"context"
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetConnectionCount() want panic  = %v, but got = %v", tt.wantPanic, got)
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewPool(tt.args.host, tt.args.port)
			got.Dial = nil
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewPool() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestAddPool(t *testing.T) {
	dbid := "dbid"
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
				for key, value := range pools {
					p[key] = value
				}

				p[dbid] = &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}
				return p
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AddPool(tt.args.dbid, tt.args.pool)
			if !reflect.DeepEqual(pools, tt.want) {
				t.Errorf("AddPool() got = %v, want = %v", pools, tt.want)
			}
		})
	}
}

func TestGetConnectionCount(t *testing.T) {
	dbid := "dbid"
	pool := NewPool("", 8080)
	pools[dbid] = &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetConnectionCount() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got, got1 := GetConnectionCount(tt.args.entityMetadata)
			if got != tt.want {
				t.Errorf("GetConnectionCount() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetConnectionCount() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func Test_getdbpool(t *testing.T) {
	dbid := "dbid"
	pool := NewPool("", 8080)
	pools[dbid] = &dbpool{ID: dbid, CtxKey: getConnectionCtxKey(dbid), Pool: pool}

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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("getdbpool() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := getdbpool(tt.args.entityMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getdbpool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetConnection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		want *redis.Pool
	}{
		{
			name: "Test_GetConnection_OK",
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConnection(); !reflect.DeepEqual(got.Pool, tt.want) {
				t.Errorf("GetConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetInfo(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		wantPanic bool
	}{
		{
			name:      "Test_GetInfo_Panic",
			wantPanic: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetInfo() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			GetInfo()
		})
	}
}

func TestGetEntityConnection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "default"
	AddPool(dbid, DefaultPool)

	type args struct {
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name string
		args args
		want *redis.Pool
	}{
		{
			name: "Test_GetEntityConnection_OK",
			args: args{entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want: DefaultPool,
		},
		{
			name: "Test_GetEntityConnection_Empty_dbid_OK",
			args: args{entityMetadata: &datastore.EntityMetadataImpl{DB: ""}},
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEntityConnection(tt.args.entityMetadata); !reflect.DeepEqual(got.Pool, tt.want) {
				t.Errorf("GetEntityConnection() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWithConnection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		args      args
		want      *redis.Pool
		wantPanic bool
	}{
		{
			name: "Test_WithConnection_Nil_Connection_In_Ctx_OK",
			args: args{ctx: context.TODO()},
			want: DefaultPool,
		},
		{
			name:      "Test_WithConnection_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 123)},
			wantPanic: true,
		},
		{
			name: "Test_WithConnection_Nil_Connection_In_Ctx_OK",
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, make(connections))},
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("WithConnection() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got := WithConnection(tt.args.ctx)
			gotCMap, ok := got.Value(CONNECTION).(connections)
			if !ok {
				t.Error("unexpected type of ctx value")
			}
			gotConn, ok := gotCMap[CONNECTION]
			if !ok {
				t.Error("expected pool in c map")
			}
			if !reflect.DeepEqual(gotConn.Pool, tt.want) {
				t.Errorf("WithConnection() = %#v, want %#v", gotConn.Pool, tt.want)
			}
		})
	}
}

func TestGetCon(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		args      args
		want      *redis.Pool
		wantPanic bool
	}{
		{
			name: "Test_GetCon_Nil_Ctx_OK",
			args: args{ctx: nil},
			want: DefaultPool,
		},
		{
			name: "Test_GetCon_Nil_Connection_Value_In_Ctx_OK",
			args: args{ctx: context.TODO()},
			want: DefaultPool,
		},
		{
			name:      "Test_GetCon_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 123)},
			wantPanic: true,
		},
		{
			name: "Test_GetCon_OK",
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, make(connections))},
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetCon() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			if got := GetCon(tt.args.ctx); !reflect.DeepEqual(got.Pool, tt.want) {
				t.Errorf("GetCon() = %v, want %v", got.Pool, tt.want)
			}
		})
	}
}

func TestWithEntityConnection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "default"
	AddPool(dbid, DefaultPool)

	anotherDbid := "another"
	anotherPool := &redis.Pool{
		Dial: DefaultPool.Dial,
	}
	AddPool(anotherDbid, anotherPool)

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name      string
		args      args
		ctxKey    common.ContextKey
		want      *redis.Pool
		wantPanic bool
	}{
		{
			name:   "Test_WithEntityConnection_DefaultPool_OK",
			args:   args{ctx: context.TODO(), entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			ctxKey: CONNECTION,
			want:   DefaultPool,
		},
		{
			name:   "Test_WithEntityConnection_Nil_Connection_Value_In_Ctx_OK",
			args:   args{ctx: context.TODO(), entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid}},
			ctxKey: getConnectionCtxKey(anotherDbid),
			want:   anotherPool,
		},
		{
			name: "Test_WithEntityConnection_OK",
			args: args{
				ctx:            context.WithValue(context.TODO(), CONNECTION, make(connections)),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			ctxKey: getConnectionCtxKey(anotherDbid),
			want:   anotherPool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("WithEntityConnection() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			got := WithEntityConnection(tt.args.ctx, tt.args.entityMetadata)
			gotCMap, ok := got.Value(CONNECTION).(connections)
			if !ok {
				t.Error("unexpected type of ctx value")
			}
			gotConn, ok := gotCMap[tt.ctxKey]
			if !ok {
				t.Error("expected pool in c map")
			}

			if !reflect.DeepEqual(gotConn.Pool, tt.want) {
				t.Errorf("WithEntityConnection() = %v, want %v", gotConn.Pool, tt.want)
			}
		})
	}
}

func TestGetEntityCon(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "default"
	AddPool(dbid, DefaultPool)

	anotherDbid := "another"
	anotherPool := &redis.Pool{
		Dial: DefaultPool.Dial,
	}
	AddPool(anotherDbid, anotherPool)

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name string
		args args
		want *redis.Pool
	}{
		{
			name: "TestGetEntityCon_Nil_Context_OK",
			args: args{entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want: DefaultPool,
		},
		{
			name: "TestGetEntityCon_Default_Pool_OK",
			args: args{ctx: context.TODO(), entityMetadata: &datastore.EntityMetadataImpl{DB: dbid}},
			want: DefaultPool,
		},
		{
			name: "TestGetEntityCon_Nil_Connection_In_Ctx_OK",
			args: args{
				ctx:            context.TODO(),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			want: nil,
		},
		{
			name: "TestGetEntityCon_OK",
			args: args{
				ctx:            context.WithValue(context.TODO(), CONNECTION, make(connections)),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			want: anotherPool,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetEntityCon(tt.args.ctx, tt.args.entityMetadata)
			if got == nil && tt.want == nil {
				return
			}
			if !reflect.DeepEqual(got.Pool, tt.want) {
				t.Errorf("GetEntityCon() = %v, want %v", got.Pool, tt.want)
			}
		})
	}
}

func TestClose(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "default"
	AddPool(dbid, DefaultPool)

	anotherDbid := "another"
	anotherPool := &redis.Pool{
		Dial: DefaultPool.Dial,
	}
	anotherConn := anotherPool.Get()
	if err := anotherConn.Close(); err != nil {
		t.Fatal(err)
	}
	AddPool(anotherDbid, anotherPool)

	cMap := connections{
		getConnectionCtxKey(dbid):        &Conn{Conn: DefaultPool.Get(), Pool: DefaultPool},
		getConnectionCtxKey(anotherDbid): &Conn{Conn: anotherConn, Pool: anotherPool},
	}

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_Close_OK",
			args: args{ctx: context.TODO()},
		},
		{
			name: "Test_Close_OK2",
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, cMap)},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Close(tt.args.ctx)
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			InitDefaultPool(tt.args.host, tt.args.port)
		})
	}
}

func Test_getConnectionCtxKey(t *testing.T) {
	t.Parallel()

	type args struct {
		dbid string
	}
	tests := []struct {
		name string
		args args
		want common.ContextKey
	}{
		{
			name: "Test_getConnectionCtxKey_Empty_Dbid_OK",
			args: args{dbid: ""},
			want: CONNECTION,
		},
		{
			args: args{dbid: "dbid"},
			want: common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, "dbid")),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := getConnectionCtxKey(tt.args.dbid); got != tt.want {
				t.Errorf("getConnectionCtxKey() = %v, want %v", got, tt.want)
			}
		})
	}
}
