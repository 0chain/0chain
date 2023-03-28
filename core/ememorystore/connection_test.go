package ememorystore

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"testing"

	"github.com/0chain/gorocksdb"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

const dataDir = "data"

func initDefaultPool() error {
	var err error
	if err := os.MkdirAll(dataDir+"/default", 0700); err != nil {
		return err
	}
	DefaultPool, err = CreateDB(dataDir + "/default")
	if err != nil {
		return err
	}

	return nil
}

func cleanUp() error {
	return os.RemoveAll(dataDir)
}

func TestCreateDB(t *testing.T) {
	t.Parallel()

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		t.Fatal(err)
	}

	bbto := gorocksdb.NewDefaultBlockBasedTableOptions()
	bbto.SetBlockCache(gorocksdb.NewLRUCache(3 << 30))
	opts := gorocksdb.NewDefaultOptions()
	opts.SetKeepLogFileNum(5)
	opts.SetBlockBasedTableFactory(bbto)
	opts.SetCreateIfMissing(true)
	tdbopts := gorocksdb.NewDefaultTransactionDBOptions()
	tDB, err := gorocksdb.OpenTransactionDb(opts, tdbopts, dataDir)
	if err != nil {
		t.Fatal(err)
	}

	tDB.Close()
	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		dataDir string
	}
	tests := []struct {
		name    string
		args    args
		want    *gorocksdb.TransactionDB
		wantErr bool
	}{
		{
			name: "Test_CreateDB_OK",
			args: args{
				dataDir: dataDir,
			},
			want:    tDB,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := CreateDB(tt.args.dataDir)
			if got != nil {
				got.Close()
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CreateDB() got = %#v, want %#v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestAddPool(t *testing.T) {
	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	type args struct {
		dbid string
		db   *gorocksdb.TransactionDB
	}
	tests := []struct {
		name string
		args args
		want *dbpool
	}{
		{
			name: "Test_AddPool_OK",
			args: args{
				dbid: dataDir,
				db:   db,
			},
			want: &dbpool{
				ID:     dataDir,
				CtxKey: common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, dataDir)),
				Pool:   db,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AddPool(tt.args.dbid, tt.args.db)
			if !reflect.DeepEqual(got, tt.want) && !reflect.DeepEqual(pools[tt.args.dbid], tt.want) {
				t.Errorf("AddPool() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestGetConnection(t *testing.T) {
	var err error
	DefaultPool, err = CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	tests := []struct {
		name string
		want *Connection
	}{
		{
			name: "Test_GetConnection_OK",
			want: GetTransaction(DefaultPool),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetConnection(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetConnection() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestGetTransaction(t *testing.T) {
	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	type args struct {
		db *gorocksdb.TransactionDB
	}
	tests := []struct {
		name string
		args args
		want *Connection
	}{
		{
			name: "Test_GetTransaction_OK",
			args: args{db: db},
			want: func() *Connection {
				ro := gorocksdb.NewDefaultReadOptions()
				wo := gorocksdb.NewDefaultWriteOptions()
				to := gorocksdb.NewDefaultTransactionOptions()

				t := db.TransactionBegin(wo, to, nil)

				return &Connection{Conn: t, ReadOptions: ro, WriteOptions: wo, TransactionOptions: to, shouldRollback: true}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetTransaction(tt.args.db); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetTransaction() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestGetEntityConnection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	em := datastore.EntityMetadataImpl{DB: "em db"}
	pools[em.DB] = &dbpool{Pool: db}

	type args struct {
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name string
		args args
		want *Connection
	}{
		{
			name: "TestGetEntityConnection_OK",
			args: args{entityMetadata: &em},
			want: func() *Connection {
				ro := gorocksdb.NewDefaultReadOptions()
				wo := gorocksdb.NewDefaultWriteOptions()
				to := gorocksdb.NewDefaultTransactionOptions()

				t := db.TransactionBegin(wo, to, nil)

				return &Connection{Conn: t, ReadOptions: ro, WriteOptions: wo, TransactionOptions: to, shouldRollback: true}
			}(),
		},
		{
			name: "TestGetEntityConnection_Empty_DB_OK",
			args: args{entityMetadata: &datastore.EntityMetadataImpl{}},
			want: GetTransaction(DefaultPool),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetEntityConnection(tt.args.entityMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityConnection() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestWithConnection(t *testing.T) {
	var err error
	DefaultPool, err = CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
		want      context.Context
	}{
		{
			name: "Test_WithConnection_OK",
			args: args{ctx: context.TODO()},
			want: func() context.Context {
				cMap := make(connections)
				cMap[CONNECTION] = GetConnection()
				return context.WithValue(context.TODO(), CONNECTION, cMap)
			}(),
		},
		{
			name:      "Test_WithConnection_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 12)},
			wantPanic: true,
		},
		{
			name: "Test_WithConnection_OK",
			args: func() args {
				cMap := make(connections)
				cMap[CONNECTION] = GetConnection()
				return args{ctx: context.WithValue(context.TODO(), CONNECTION, cMap)}
			}(),
			want: func() context.Context {
				cMap := make(connections)
				cMap[CONNECTION] = GetConnection()
				return context.WithValue(context.TODO(), CONNECTION, cMap)
			}(),
		},
		{
			name: "Test_WithConnection_Default_Conn_OK",
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, make(connections))},
			want: func() context.Context {
				cMap := make(connections)
				cMap[CONNECTION] = GetConnection()
				return context.WithValue(context.TODO(), CONNECTION, cMap)
			}(),
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
			if got := WithConnection(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithConnection() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestGetEntityCon(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	em := datastore.EntityMetadataImpl{DB: "em db"}
	pools[em.DB] = &dbpool{Pool: db}

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name      string
		args      args
		want      *Connection
		wantPanic bool
	}{
		{
			name: "Test_GetEntityCon_Nil_Ctx_OK",
			args: args{ctx: nil, entityMetadata: &em},
			want: GetEntityConnection(&em),
		},
		{
			name: "Test_GetEntityCon_Default_Pool_OK",
			args: func() args {
				db := "default"
				pools[db] = &dbpool{Pool: DefaultPool}

				return args{ctx: context.TODO(), entityMetadata: &datastore.EntityMetadataImpl{DB: db}}
			}(),
			want: GetCon(context.TODO()),
		},
		{
			name: "Test_GetEntityCon_Nil_Connection_In_Context_OK",
			args: args{ctx: context.TODO(), entityMetadata: &em},
			want: nil,
		},
		{
			name:      "Test_GetEntityCon_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 12), entityMetadata: &em},
			wantPanic: true,
		},
		{
			name: "Test_GetEntityCon_Empty_Conn_OK",
			args: func() args {
				ctxKey := common.ContextKey("ctxKey")
				pools[em.DB] = &dbpool{Pool: db, CtxKey: ctxKey}

				cMap := make(connections)

				return args{
					ctx:            context.WithValue(context.TODO(), CONNECTION, cMap),
					entityMetadata: &datastore.EntityMetadataImpl{DB: em.DB},
				}
			}(),
			want: GetConnection(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("GetEntityCon() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()
			if got := GetEntityCon(tt.args.ctx, tt.args.entityMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntityCon() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestGetCon(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		args      args
		want      *Connection
		wantPanic bool
	}{
		{
			name: "Test_GetCon_Nil_Ctx_OK",
			args: args{ctx: nil},
			want: GetConnection(),
		},
		{
			name: "Test_GetCon_Nil_Connection_In_Context_OK",
			args: args{ctx: context.TODO()},
			want: GetConnection(),
		},
		{
			name:      "Test_GetCon_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 12)},
			wantPanic: true,
		},
		{
			name: "Test_GetCon_Empty_Conn_OK",
			args: args{ctx: context.WithValue(context.TODO(), CONNECTION, make(connections))},
			want: GetConnection(),
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

			if got := GetCon(tt.args.ctx); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetCon() = %v, want %v", got, tt.want)
			}
		})
	}
	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestWithEntityConnection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	em := datastore.EntityMetadataImpl{DB: "em db"}
	pools[em.DB] = &dbpool{Pool: db}

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
	}
	tests := []struct {
		name      string
		args      args
		want      context.Context
		wantPanic bool
	}{
		{
			name: "Test_WithEntityConnection_Default_Pool_OK",
			args: func() args {
				db := "default"
				pools[db] = &dbpool{Pool: DefaultPool}

				return args{ctx: context.TODO(), entityMetadata: &datastore.EntityMetadataImpl{DB: db}}
			}(),
			want: WithConnection(context.TODO()),
		},
		{
			name: "Test_WithEntityConnection_Nil_Connection_In_Context_OK",
			args: func() args {
				ctxKey := common.ContextKey("ctxKey")
				pools[em.DB] = &dbpool{Pool: db, CtxKey: ctxKey}

				return args{ctx: context.TODO(), entityMetadata: &datastore.EntityMetadataImpl{DB: em.DB}}
			}(),
			want: func() context.Context {
				ctxKey := common.ContextKey("ctxKey")
				cMap := make(connections)
				cMap[ctxKey] = GetTransaction(db)
				return context.WithValue(context.TODO(), CONNECTION, cMap)
			}(),
		},
		{
			name:      "Test_WithEntityConnection_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 12), entityMetadata: &em},
			wantPanic: true,
		},
		{
			name: "Test_GetEntityCon_Empty_Conn_OK",
			args: func() args {
				ctxKey := common.ContextKey("ctxKey")
				pools[em.DB] = &dbpool{Pool: db, CtxKey: ctxKey}

				cMap := make(connections)

				return args{
					ctx:            context.WithValue(context.TODO(), CONNECTION, cMap),
					entityMetadata: &datastore.EntityMetadataImpl{DB: em.DB},
				}
			}(),
			want: func() context.Context {
				ctxKey := common.ContextKey("ctxKey")
				cMap := make(connections)
				cMap[ctxKey] = GetTransaction(db)
				return context.WithValue(context.TODO(), CONNECTION, cMap)
			}(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("WithEntityConnection() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()
			if got := WithEntityConnection(tt.args.ctx, tt.args.entityMetadata); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WithEntityConnection() = %v, want %v", got, tt.want)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestClose(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}
	defer DefaultPool.Close()

	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		args      args
		wantPanic bool
	}{
		{
			name: "Test_Close_No_Value_In_Ctx_OK",
			args: args{ctx: context.TODO()},
		},
		{
			name:      "Test_Close_Panic",
			args:      args{ctx: context.WithValue(context.TODO(), CONNECTION, 123)},
			wantPanic: true,
		},
		{
			name: "Test_Close_Panic",
			args: func() args {
				ctxKey := common.ContextKey("ctxKey")

				cMap := make(connections)
				cMap[ctxKey] = GetConnection()

				return args{
					ctx: context.WithValue(context.TODO(), CONNECTION, cMap),
				}
			}(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				got := recover()
				if (got != nil) != tt.wantPanic {
					t.Errorf("Close() want panic  = %v, but got = %v", tt.wantPanic, got)
				}
			}()

			Close(tt.args.ctx)
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
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
			name: "Test_getConnectionCtxKey_Emtpy_dbid_OK",
			args: args{dbid: ""},
			want: CONNECTION,
		},
		{
			name: "Test_getConnectionCtxKey_OK",
			args: args{dbid: "123"},
			want: common.ContextKey(fmt.Sprintf("%v%v", CONNECTION, "123")),
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

func Test_getdbpool(t *testing.T) {
	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	em := datastore.EntityMetadataImpl{DB: "em db"}
	pools[em.DB] = &dbpool{Pool: db}

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
			name:      "Test_getdbpool_Panic",
			args:      args{entityMetadata: &datastore.EntityMetadataImpl{}},
			wantPanic: true,
		},
		{
			name: "Test_getdbpool_OK",
			args: args{entityMetadata: &em},
			want: pools[em.DB],
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}

func TestConnection_Commit(t *testing.T) {
	db, err := CreateDB(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	conn := GetTransaction(db)

	type fields struct {
		Conn               *gorocksdb.Transaction
		ReadOptions        *gorocksdb.ReadOptions
		WriteOptions       *gorocksdb.WriteOptions
		TransactionOptions *gorocksdb.TransactionOptions
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "TestConnection_Commit_OK",
			fields: fields{
				Conn:               conn.Conn,
				ReadOptions:        conn.ReadOptions,
				WriteOptions:       conn.WriteOptions,
				TransactionOptions: conn.TransactionOptions,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Connection{
				Conn:               tt.fields.Conn,
				ReadOptions:        tt.fields.ReadOptions,
				WriteOptions:       tt.fields.WriteOptions,
				TransactionOptions: tt.fields.TransactionOptions,
			}
			if err := c.Commit(); (err != nil) != tt.wantErr {
				t.Errorf("Commit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}

	if err := cleanUp(); err != nil {
		t.Fatal(err)
	}
}
