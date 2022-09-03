package memorystore

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"github.com/0chain/common/core/logging"
	"github.com/gomodule/redigo/redis"
)

func init() {
	logging.InitLogging("development", "")
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
		{
			name: "Test_GetConnection_OK",
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
		{
			name:      "Test_GetInfo_Panic",
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
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, newConnections())},
			want: DefaultPool,
		},
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
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, newConnections())},
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
			gotConn, ok := gotCMap.get(CONNECTION)
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
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, newConnections())},
			want: DefaultPool,
		},
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
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, newConnections())},
			want: DefaultPool,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
				ctx:            context.WithValue(context.TODO(), CONNECTION, newConnections()),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			ctxKey: getConnectionCtxKey(anotherDbid),
			want:   anotherPool,
		},
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
				ctx:            context.WithValue(context.TODO(), CONNECTION, newConnections()),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			ctxKey: getConnectionCtxKey(anotherDbid),
			want:   anotherPool,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
			gotConn, ok := gotCMap.get(tt.ctxKey)
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
				ctx:            context.WithValue(context.TODO(), CONNECTION, newConnections()),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			want: anotherPool,
		},
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
				ctx:            context.WithValue(context.TODO(), CONNECTION, newConnections()),
				entityMetadata: &datastore.EntityMetadataImpl{DB: anotherDbid},
			},
			want: anotherPool,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
		cons: map[common.ContextKey]*Conn{
			getConnectionCtxKey(dbid):        &Conn{Conn: DefaultPool.Get(), Pool: DefaultPool},
			getConnectionCtxKey(anotherDbid): &Conn{Conn: anotherConn, Pool: anotherPool},
		},
		mutex: &sync.RWMutex{},
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
		{
			name: "Test_Close_OK2",
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, cMap)},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			Close(tt.args.ctx)
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
