package ememorystore_test

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"testing"

	"github.com/0chain/gorocksdb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/encryption"
)

const dataDir = "data"

func init() {
	sp := ememorystore.GetStorageProvider()
	round.SetupEntity(sp)
	block.SetupEntity(sp)
	em := block.Provider().GetEntityMetadata().(*datastore.EntityMetadataImpl)
	em.DB = "block"
}

// connections stores keys as entity name and values as database name
var connections = map[string]string{
	"round": "roundsummarydb",
	"block": "block",
}

var pools = make(map[string]*gorocksdb.TransactionDB)

func initDBs() error {
	for entityName, dbName := range connections {
		dir := dataDir + "/" + dbName
		err := os.MkdirAll(dir, 0700)
		if err != nil {
			return err
		}
		db, err := ememorystore.CreateDB(dir)
		if err != nil {
			return err
		}

		ememorystore.AddPool(dbName, db)

		_, ok := pools[entityName]
		if ok {
			return errors.New("trying to init existing db")
		}
		pools[entityName] = db
	}

	return nil
}

func closeTestPools() {
	for _, pool := range pools {
		pool.Close()
	}
	pools = make(map[string]*gorocksdb.TransactionDB)
}

func TestMain(m *testing.M) {
	if err := initDBs(); err != nil {
		panic(err)
	}

	r := m.Run()

	closeTestPools()
	if err := cleanUp(); err != nil {
		panic(err)
	}

	os.Exit(r)
}

func cleanUp() error {
	return os.RemoveAll(dataDir)
}

func TestStore_Read(t *testing.T) {
	r := round.NewRound(1)
	r.BlockHash = encryption.Hash("data")
	rByt, err := json.Marshal(r)
	if err != nil {
		t.Error(err)
	}
	ctx := ememorystore.WithEntityConnection(context.TODO(), r.GetEntityMetadata())
	txn := ememorystore.GetEntityCon(ctx, r.GetEntityMetadata())
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, 1)
	if err := txn.Conn.Put(key, rByt); err != nil {
		t.Fatal(err)
	}
	if err := txn.Conn.Commit(); err != nil {
		t.Fatal(err)
	}

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()
	bByt, err := json.Marshal(b)
	if err != nil {
		t.Error(err)
	}

	ctx = ememorystore.WithEntityConnection(context.TODO(), b.GetEntityMetadata())
	txn = ememorystore.GetEntityCon(ctx, b.GetEntityMetadata())
	if err := txn.Conn.Put([]byte(b.GetKey()), bByt); err != nil {
		t.Fatal(err)
	}
	// put invalid json data
	invK := []byte("inv")
	if err := txn.Conn.Put(invK, []byte("}{")); err != nil {
		t.Fatal(err)
	}
	if err := txn.Conn.Commit(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx    context.Context
		key    datastore.Key
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    datastore.Entity
		wantErr bool
	}{
		{
			name:    "Test_Store_Read_Round_OK",
			args:    args{key: r.GetKey(), entity: round.NewRound(1)},
			want:    r,
			wantErr: false,
		},
		{
			name:    "Test_Store_Read_Block_OK",
			args:    args{key: b.GetKey(), entity: b},
			want:    b,
			wantErr: false,
		},
		{
			name:    "Test_Store_Read_JSON_ERR",
			args:    args{key: string(invK), entity: block.NewBlock("", 1)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ems := &ememorystore.Store{}
			if err := ems.Read(tt.args.ctx, tt.args.key, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				assert.Equal(t, tt.want, tt.args.entity)
			}
		})
	}
}

func refreshDBs() error {
	closeTestPools()
	if err := cleanUp(); err != nil {
		return err
	}

	if err := initDBs(); err != nil {
		return err
	}

	return nil
}

func TestStore_Write(t *testing.T) {
	r := round.NewRound(1)
	r.BlockHash = encryption.Hash("data")

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Store_Write_Round_OK",
			args:    args{entity: r},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Block_OK",
			args:    args{entity: block.NewBlock("", 1)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := refreshDBs(); err != nil {
				t.Error(err)
			}

			ems := &ememorystore.Store{}
			if err := ems.Write(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Delete(t *testing.T) {
	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestStore_Delete_OK",
			args:    args{entity: block.NewBlock("", 1)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := refreshDBs(); err != nil {
				t.Error(err)
			}

			ems := &ememorystore.Store{}
			if err := ems.Delete(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiRead(t *testing.T) {
	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()
	bByt, err := json.Marshal(b)
	if err != nil {
		t.Error(err)
	}

	ctx := ememorystore.WithEntityConnection(context.TODO(), b.GetEntityMetadata())
	txn := ememorystore.GetEntityCon(ctx, b.GetEntityMetadata())
	if err := txn.Conn.Put([]byte(b.GetKey()), bByt); err != nil {
		t.Fatal(err)
	}
	// put invalid json data
	invK := []byte("inv")
	if err := txn.Conn.Put(invK, []byte("}{")); err != nil {
		t.Fatal(err)
	}
	if err := txn.Conn.Commit(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		keys           []datastore.Key
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		want    []datastore.Entity
		wantErr bool
	}{
		{
			name: "TestStore_MultiRead_OK",
			args: args{
				entityMetadata: b.GetEntityMetadata(),
				keys: []datastore.Key{
					b.GetKey(),
					string(invK),
				},
				entities: []datastore.Entity{
					b,
					block.NewBlock("", 1),
				},
			},
			want: func() []datastore.Entity {
				invB := block.NewBlock("", 1)
				b.SetKey(datastore.EmptyKey)

				return []datastore.Entity{
					b,
					invB,
				}
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ems := &ememorystore.Store{}
			if err := ems.MultiRead(tt.args.ctx, tt.args.entityMetadata, tt.args.keys, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiRead() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr && !reflect.DeepEqual(tt.args.entities, tt.want) {
				t.Errorf("Read() got = %v, want = %v", tt.args.entities, tt.want)
			}
		})
	}
}

func TestStore_MultiWrite(t *testing.T) {
	b1 := block.NewBlock("", 1)
	b1.Hash = b1.ComputeHash()
	b2 := block.NewBlock("", 2)
	b2.Hash = b2.ComputeHash()

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestStore_MultiWrite_OK",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					b1,
					b2,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := refreshDBs(); err != nil {
				t.Error(err)
			}

			ems := &ememorystore.Store{}
			if err := ems.MultiWrite(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDelete(t *testing.T) {
	b1 := block.NewBlock("", 1)
	b1.Hash = b1.ComputeHash()
	b2 := block.NewBlock("", 2)
	b2.Hash = b2.ComputeHash()

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "TestStore_MultiDelete_OK",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					b1,
					b2,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := refreshDBs()
			require.NoError(t, err)

			ems := &ememorystore.Store{}
			if err := ems.MultiDelete(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_AddToCollection(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx    context.Context
		entity datastore.CollectionEntity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestStore_AddToCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if err := ems.AddToCollection(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("AddToCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiAddToCollection(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestStore_MultiAddToCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if err := ems.MultiAddToCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiAddToCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_DeleteFromCollection(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx    context.Context
		entity datastore.CollectionEntity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestStore_DeleteFromCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if err := ems.DeleteFromCollection(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("DeleteFromCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDeleteFromCollection(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Store_MultiDeleteFromCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if err := ems.MultiDeleteFromCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiDeleteFromCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_GetCollectionSize(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		collectionName string
	}
	tests := []struct {
		name string
		args args
		want int64
	}{
		{
			name: "Test_Store_GetCollectionSize_OK",
			want: -1, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if got := ems.GetCollectionSize(tt.args.ctx, tt.args.entityMetadata, tt.args.collectionName); got != tt.want {
				t.Errorf("GetCollectionSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_IterateCollection(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		collectionName string
		handler        datastore.CollectionIteratorHandler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Store_IterateCollection_OK",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if err := ems.IterateCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.collectionName, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("IterateCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_InsertIfNE(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx    context.Context
		entity datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "Test_Store_InsertIfNE_OK",
			args:    args{entity: block.NewBlock("", 1)},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ems := &ememorystore.Store{}
			if err := ems.InsertIfNE(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("InsertIfNE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
