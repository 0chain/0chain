package memorystore_test

import (
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
	"context"
	"testing"
	"time"
)

func init() {
	common.SetupRootContext(node.GetNodeContext())

	transaction.SetupEntity(memorystore.GetStorageProvider())
}

func TestStore_IterateCollection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "txndb"
	memorystore.AddPool(dbid, memorystore.DefaultPool)

	handler := func(ctx context.Context, ce datastore.CollectionEntity) bool {
		if ce.GetEntityMetadata().GetName() == "txn" {
			ent := ce.(*transaction.Transaction)
			if ent.Value > 5 {
				ent.Value++
				return true
			}
			return false
		}
		return false
	}

	txn := transaction.Transaction{}
	txn.SetKey("key")
	sp := memorystore.GetStorageProvider()
	if err := sp.Write(nil, &txn); err != nil {
		t.Fatal(err)
	}
	if err := sp.AddToCollection(nil, &txn); err != nil {
		t.Fatal(err)
	}
	txn2 := transaction.Transaction{}
	txn2.SetKey("key2")
	if err := sp.Write(nil, &txn2); err != nil {
		t.Fatal(err)
	}
	if err := sp.AddToCollection(nil, &txn2); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entityMetadata datastore.EntityMetadata
		collectionName string
		handler        datastore.CollectionIteratorHandler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		cancel  bool
	}{
		{
			name: "Test_Store_IterateCollection_Empty_Data_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				handler:        handler,
			},
			wantErr: false,
		},
		{
			name: "Test_Store_IterateCollection_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
				handler:        handler,
			},
			wantErr: false,
		},
		{
			name: "Test_Store_IterateCollection_Cancel_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
				handler:        handler,
			},
			cancel:  true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &memorystore.Store{}
			ctx, cancel := context.WithCancel(context.TODO())

			if tt.cancel {
				cancel()
			}

			if err := ms.IterateCollection(ctx, tt.args.entityMetadata, tt.args.collectionName, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("IterateCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_IterateCollection_Closed_Conn_Err(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "txndb"
	memorystore.AddPool(dbid, memorystore.DefaultPool)

	handler := func(ctx context.Context, ce datastore.CollectionEntity) bool {
		if ce.GetEntityMetadata().GetName() == "txn" {
			ent := ce.(*transaction.Transaction)
			if ent.Value > 5 {
				ent.Value++
				return true
			}
			return false
		}
		return false
	}

	txn := transaction.Transaction{}
	txn.SetKey("key")
	sp := memorystore.GetStorageProvider()
	if err := sp.Write(nil, &txn); err != nil {
		t.Fatal(err)
	}
	if err := sp.AddToCollection(nil, &txn); err != nil {
		t.Fatal(err)
	}
	txn2 := transaction.Transaction{}
	txn2.SetKey("key2")
	if err := sp.Write(nil, &txn2); err != nil {
		t.Fatal(err)
	}
	if err := sp.AddToCollection(nil, &txn2); err != nil {
		t.Fatal(err)
	}

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

	type args struct {
		entityMetadata datastore.EntityMetadata
		collectionName string
		handler        datastore.CollectionIteratorHandler
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		cancel  bool
	}{
		{
			name: "Test_Store_IterateCollection_Empty_Data_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				handler:        handler,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &memorystore.Store{}
			ctx, cancel := context.WithCancel(context.TODO())

			if tt.cancel {
				cancel()
			}

			if err := ms.IterateCollection(ctx, tt.args.entityMetadata, tt.args.collectionName, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("IterateCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_IterateCollectionAsc(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "txndb"
	memorystore.AddPool(dbid, memorystore.DefaultPool)

	handler := func(ctx context.Context, ce datastore.CollectionEntity) bool {
		if ce.GetEntityMetadata().GetName() == "txn" {
			ent := ce.(*transaction.Transaction)
			if ent.Value > 5 {
				ent.Value++
				return true
			}
			return false
		}
		return false
	}

	txn := transaction.Transaction{}
	txn.SetKey("key")
	sp := memorystore.GetStorageProvider()
	if err := sp.Write(nil, &txn); err != nil {
		t.Fatal(err)
	}
	if err := sp.AddToCollection(nil, &txn); err != nil {
		t.Fatal(err)
	}

	type args struct {
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
			name: "Test_Store_IterateCollectionAsc_Empty_Data_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				handler:        handler,
			},
			wantErr: false,
		},
		{
			name: "Test_Store_IterateCollectionAsc_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
				handler:        handler,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ms := &memorystore.Store{}
			ctx, _ := context.WithCancel(context.TODO())

			if err := ms.IterateCollectionAsc(ctx, tt.args.entityMetadata, tt.args.collectionName, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("IterateCollectionAsc() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPrintIterator(t *testing.T) {
	t.Parallel()

	type args struct {
		ctx context.Context
		qe  datastore.CollectionEntity
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "Test_PrintIterator_OK",
			args: args{ctx: nil, qe: &transaction.Transaction{}},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := memorystore.PrintIterator(tt.args.ctx, tt.args.qe); got != tt.want {
				t.Errorf("PrintIterator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCollectionTrimmer(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	dbid := "txndb"
	memorystore.AddPool(dbid, memorystore.DefaultPool)

	txn := transaction.Transaction{}

	type args struct {
		entityMetadata datastore.EntityMetadata
		collection     string
		trimSize       int64
		trimBeyond     time.Duration
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Test_CollectionTrimmer_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				collection:     txn.GetCollectionName(),
				trimBeyond:     time.Nanosecond,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go func() {
				time.Sleep(time.Millisecond * 20)
				common.Done()
			}()
			memorystore.CollectionTrimmer(tt.args.entityMetadata, tt.args.collection, tt.args.trimSize, tt.args.trimBeyond)
		})
	}
}
