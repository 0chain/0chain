package memorystore_test

import (
	"context"
	"fmt"

	"0chain.net/chaincore/block"
	"0chain.net/chaincore/node"
	"0chain.net/chaincore/round"
	"0chain.net/chaincore/transaction"
	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/encryption"
	"0chain.net/core/memorystore"
	"github.com/alicebob/miniredis/v2"
	"github.com/gomodule/redigo/redis"
	"github.com/stretchr/testify/require"
	"github.com/vmihailenco/msgpack/v5"

	"reflect"
	"testing"
	"time"
)

func init() {
	common.SetupRootContext(node.GetNodeContext())

	block.SetupEntity(memorystore.GetStorageProvider())
	transaction.SetupEntity(memorystore.GetStorageProvider())
	round.SetupEntity(memorystore.GetStorageProvider())
}

func initDefaultPool() error {
	mr, err := miniredis.Run()
	if err != nil {
		return err
	}

	memorystore.DefaultPool = &redis.Pool{
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

func TestStore_Read(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()
	if err := memorystore.GetStorageProvider().Write(nil, b); err != nil {
		t.Fatal()
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
			name:    "Test_Store_Read_OK",
			args:    args{ctx: nil, key: b.GetKey(), entity: block.Provider()},
			want:    b,
			wantErr: false,
		},
		{
			name:    "Test_Store_Read_Nil_Data_ERR",
			args:    args{ctx: nil, entity: block.Provider()},
			want:    b,
			wantErr: true,
		},
		{
			name:    "Test_Store_Read_OK",
			args:    args{ctx: nil, key: b.GetKey(), entity: block.Provider()},
			want:    b,
			wantErr: false,
		},
		{
			name:    "Test_Store_Read_Nil_Data_ERR",
			args:    args{ctx: nil, entity: block.Provider()},
			want:    b,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.Read(tt.args.ctx, tt.args.key, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.args.entity.GetKey(), tt.want.GetKey()) {
				t.Errorf("Read() got = %v, want = %v", tt.args.entity, tt.want)
			}
		})
	}
}

func TestStore_Read_Closed_Conn_ERR(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	if err := memorystore.DefaultPool.Close(); err != nil {
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
			name:    "Test_Store_Read_Closed_Conn_ERR",
			args:    args{ctx: nil, key: b.GetKey(), entity: block.Provider()},
			wantErr: true,
		},
		{
			name:    "Test_Store_Read_Closed_Conn_ERR",
			args:    args{ctx: nil, key: b.GetKey(), entity: block.Provider()},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.Read(tt.args.ctx, tt.args.key, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.args.entity.GetKey(), tt.want.GetKey()) {
				t.Errorf("Read() got = %v, want = %v", tt.args.entity, tt.want)
			}
		})
	}
}

func TestStore_Write(t *testing.T) {
	initDefaultTxnPool(t)

	// No sure if this configuration is needed for this test!
	// config.DevConfiguration.IsFeeEnabled = true

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	txn := transaction.Transaction{}
	txn.SetCollectionScore(1)

	txn2 := transaction.Transaction{}
	txn2.Fee = 1

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
			name:    "Test_Store_Write_Block_OK",
			args:    args{ctx: nil, entity: b},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Txn_OK",
			args:    args{ctx: nil, entity: &transaction.Transaction{}},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Txn_Non_Empty_Score_OK",
			args:    args{ctx: nil, entity: &txn},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Txn_Non_Empty_Score_And_Score_OK",
			args:    args{ctx: nil, entity: &txn2},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Block_OK",
			args:    args{ctx: nil, entity: b},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Txn_OK",
			args:    args{ctx: nil, entity: &transaction.Transaction{}},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Txn_Non_Empty_Score_OK",
			args:    args{ctx: nil, entity: &txn},
			wantErr: false,
		},
		{
			name:    "Test_Store_Write_Txn_Non_Empty_Score_And_Score_OK",
			args:    args{ctx: nil, entity: &txn2},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.Write(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Write_Closed_Conn_ERR(t *testing.T) {
	initDefaultTxnPool(t)

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

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
			name:    "Test_Store_Write_Closed_Conn_OK",
			args:    args{ctx: nil, entity: b},
			wantErr: true,
		},
		{
			name:    "Test_Store_Write_Closed_Conn_OK",
			args:    args{ctx: nil, entity: b},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.Write(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_InsertIfNE(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	if err := memorystore.GetStorageProvider().Write(nil, b); err != nil {
		t.Fatal(err)
	}

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
			name:    "Test_Store_InsertIfNE_ERR",
			args:    args{entity: b},
			wantErr: true,
		},
		{
			name:    "Test_Store_InsertIfNE_ERR",
			args:    args{entity: b},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.InsertIfNE(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("InsertIfNE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Delete(t *testing.T) {
	initDefaultTxnPool(t)

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
			name:    "Test_Store_Delete_Txn_OK",
			args:    args{entity: &transaction.Transaction{}},
			wantErr: false,
		},
		{
			name:    "Test_Store_Delete_Block_OK",
			args:    args{entity: block.NewBlock("", 1)},
			wantErr: false,
		},
		{
			name:    "Test_Store_Delete_Txn_OK",
			args:    args{entity: &transaction.Transaction{}},
			wantErr: false,
		},
		{
			name:    "Test_Store_Delete_Block_OK",
			args:    args{entity: block.NewBlock("", 1)},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.Delete(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Delete_Closed_Conn_ERR(t *testing.T) {
	initDefaultTxnPool(t)

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

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
			name:    "Test_Store_Delete_Closed_Conn_ERR",
			args:    args{entity: &transaction.Transaction{}},
			wantErr: true,
		},
		{
			name:    "Test_Store_Delete_Closed_Conn_ERR",
			args:    args{entity: &transaction.Transaction{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.Delete(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiRead(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	memorystore.AddPool("roundsummarydb", memorystore.DefaultPool)

	r := round.NewRound(1)
	r2 := round.NewRound(2)

	if err := memorystore.GetStorageProvider().Write(nil, r); err != nil {
		t.Fatal(err)
	}
	if err := memorystore.GetStorageProvider().Write(nil, r2); err != nil {
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
		wantErr bool
	}{
		{
			name: "Test_Store_MultiRead_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					r.GetKey(),
					r2.GetKey(),
				},
				entities: []datastore.Entity{
					round.Provider(),
					round.Provider(),
				},
			},
		},
		{
			name: "Test_Store_MultiRead_Size>256_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = round.Provider()
					}

					return en
				}(),
			},
		},
		{
			name: "Test_Store_MultiRead_Zero_Size_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
			},
		},
		{
			name: "Test_Store_MultiRead_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					r.GetKey(),
					r2.GetKey(),
				},
				entities: []datastore.Entity{
					round.Provider(),
					round.Provider(),
				},
			},
		},
		{
			name: "Test_Store_MultiRead_Size>256_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = round.Provider()
					}

					return en
				}(),
			},
		},
		{
			name: "Test_Store_MultiRead_Zero_Size_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiRead(tt.args.ctx, tt.args.entityMetadata, tt.args.keys, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiRead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiRead_Closed_Conn_ERR(t *testing.T) {
	initDefaultTxnPool(t)

	txn := transaction.Transaction{}
	txn.SetKey("key")
	sp := memorystore.GetStorageProvider()
	if err := sp.Write(nil, &txn); err != nil {
		t.Fatal(err)
	}

	if err := memorystore.DefaultPool.Close(); err != nil {
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
		wantErr bool
	}{
		{
			name: "Test_Store_MultiRead_Closed_Conn_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					txn.GetKey(),
				},
				entities: []datastore.Entity{
					transaction.Provider(),
				},
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiRead_Size>256_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = transaction.Provider()
					}

					return en
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiRead_Closed_Conn_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					txn.GetKey(),
				},
				entities: []datastore.Entity{
					transaction.Provider(),
				},
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiRead_Size>256_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = transaction.Provider()
					}

					return en
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiRead(tt.args.ctx, tt.args.entityMetadata, tt.args.keys, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiRead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})

	}
}

func TestStore_MultiWrite(t *testing.T) {
	initDefaultTxnPool(t)

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")

	txn2 := transaction.Transaction{}
	txn2.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn2.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn2.SetKey("key2")

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		keys           []datastore.Key
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Test_Store_MultiWrite_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					txn.GetKey(),
					txn2.GetKey(),
				},
				entities: []datastore.Entity{
					&txn,
					&txn2,
				},
			},
		},
		{
			name: "Test_Store_MultiWrite_Size>256_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = round.Provider()
					}

					return en
				}(),
			},
		}, {
			name: "Test_Store_MultiWrite_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					txn.GetKey(),
					txn2.GetKey(),
				},
				entities: []datastore.Entity{
					&txn,
					&txn2,
				},
			},
		},
		{
			name: "Test_Store_MultiWrite_Size>256_OK",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = round.Provider()
					}

					return en
				}(),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiWrite(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiWrite_Closed_Conn_ERR(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	memorystore.AddPool("roundsummarydb", memorystore.DefaultPool)

	if err := memorystore.DefaultPool.Close(); err != nil {
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
		wantErr bool
	}{
		{
			name: "Test_Store_MultiWrite_Closed_Conn_ERR",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = round.Provider()
					}

					return en
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiWrite_Closed_Conn_ERR",
			args: args{
				entityMetadata: round.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, 257),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = round.Provider()
					}

					return en
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiWrite(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDelete(t *testing.T) {
	initDefaultTxnPool(t)

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")

	txn2 := transaction.Transaction{}
	txn2.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn2.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn2.SetKey("key2")

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
			name: "Test_Store_MultiDelete_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					&txn2,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDelete_Size>256_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = transaction.Provider()
					}

					return en
				}(),
			},
		},
		{
			name: "Test_Store_MultiDelete_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					&txn2,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDelete_Size>256_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = transaction.Provider()
					}

					return en
				}(),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiDelete(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDelete_Closed_Conn_ERR(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

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
			name: "Test_Store_MultiDelete_Closed_Conn_ERR",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = block.Provider()
					}

					return en
				}(),
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiDelete_Closed_Conn_ERR",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: func() []datastore.Entity {
					en := make([]datastore.Entity, 257)
					for ind := range en {
						en[ind] = block.Provider()
					}

					return en
				}(),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiDelete(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiAddToCollection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	memorystore.AddPool("txndb", memorystore.DefaultPool)
	memorystore.AddPool("roundsummarydb", memorystore.DefaultPool)

	//config.DevConfiguration.IsFeeEnabled = true

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.SetCollectionScore(0)
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Hour
	txn.SetKey("key")

	txn2 := transaction.Transaction{}
	txn2.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn2.SetCollectionScore(0)
	txn2.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn2.SetKey("key")
	txn2.Fee = 1

	txn3 := transaction.Transaction{}
	txn3.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn3.SetCollectionScore(0)
	txn3.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn3.SetKey("key")
	txn3.Fee = 1

	writeTxnsToStorage(t, &txn, &txn2, &txn3)

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
			name: "Test_Store_MultiAddToCollection_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					&txn2,
					&txn3,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiAddToCollection_Size>256_OK",
			args: func() args {
				entities := make([]datastore.Entity, 128)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					sp := memorystore.GetStorageProvider()
					if err := sp.Write(nil, &txn); err != nil {
						t.Fatal(err)
					}
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: false,
		},
		{
			name: "Test_Store_MultiAddToCollection_Zero_Size_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities:       make([]datastore.Entity, 0),
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiAddToCollection_Not_Collection_Entity_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					&round.Round{},
				},
			},
			wantErr: true,
		}, {
			name: "Test_Store_MultiAddToCollection_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					&txn2,
					&txn3,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiAddToCollection_Size>256_OK",
			args: func() args {
				entities := make([]datastore.Entity, 128)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					sp := memorystore.GetStorageProvider()
					if err := sp.Write(nil, &txn); err != nil {
						t.Fatal(err)
					}
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: false,
		},
		{
			name: "Test_Store_MultiAddToCollection_Zero_Size_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities:       make([]datastore.Entity, 0),
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiAddToCollection_Not_Collection_Entity_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					&round.Round{},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiAddToCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiAddToCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiAddToCollection_Closed_Conn_ERR(t *testing.T) {
	initDefaultTxnPool(t)

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

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
			name: "Test_Store_MultiAddToCollection_Size>256_OK",
			args: func() args {
				entities := make([]datastore.Entity, 257)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Test_Store_MultiAddToCollection_Size>256_OK",
			args: func() args {
				entities := make([]datastore.Entity, 257)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiAddToCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiAddToCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDeleteFromCollection(t *testing.T) {
	if err := initDefaultPool(); err != nil {
		t.Fatal(err)
	}

	memorystore.AddPool("txndb", memorystore.DefaultPool)
	memorystore.AddPool("roundsummarydb", memorystore.DefaultPool)

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")
	writeTxnsToStorage(t, &txn)

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
			name: "Test_Store_MultiDeleteFromCollection_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					round.Provider(),
				},
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_Zero_Size_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities:       []datastore.Entity{},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_Size>256_OK",
			args: func() args {
				entities := make([]datastore.Entity, 128)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					sp := memorystore.GetStorageProvider()
					if err := sp.Write(nil, &txn); err != nil {
						t.Fatal(err)
					}
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_ERR",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					&txn,
					round.Provider(),
				},
			},
			wantErr: true,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_Zero_Size_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				entities:       []datastore.Entity{},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_Size>256_OK",
			args: func() args {
				entities := make([]datastore.Entity, 128)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					sp := memorystore.GetStorageProvider()
					if err := sp.Write(nil, &txn); err != nil {
						t.Fatal(err)
					}
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiDeleteFromCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiDeleteFromCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDeleteFromCollection_Closed_Conn_ERR(t *testing.T) {
	initDefaultTxnPool(t)

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

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
			name: "Test_Store_MultiDeleteFromCollection_Closed_Conn_ERR",
			args: func() args {
				entities := make([]datastore.Entity, 257)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: true,
		},
		{
			name: "Test_Store_MultiDeleteFromCollection_Closed_Conn_ERR",
			args: func() args {
				entities := make([]datastore.Entity, 257)
				for ind := range entities {
					txn := transaction.Transaction{}
					txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
					txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
					txn.SetKey(fmt.Sprintf("key:%v", ind))
					entities[ind] = &txn
				}

				return args{
					entityMetadata: transaction.Provider().GetEntityMetadata(),
					entities:       entities,
				}
			}(),
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.MultiDeleteFromCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiDeleteFromCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_GetCollectionSize(t *testing.T) {
	initDefaultTxnPool(t)

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")
	writeTxnsToStorage(t, &txn)
	addTxnsToCollection(t, &txn)

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
			args: args{
				entityMetadata: txn.GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
			},
			want: 1,
		},
		{
			name: "Test_Store_GetCollectionSize_OK",
			args: args{
				entityMetadata: txn.GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if got := ms.GetCollectionSize(tt.args.ctx, tt.args.entityMetadata, tt.args.collectionName); got != tt.want {
				t.Errorf("GetCollectionSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_GetCollectionSiz_Closed_Conn(t *testing.T) {
	initDefaultTxnPool(t)

	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

	txn := transaction.Transaction{}

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
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
			},
			want: -1,
		},
		{
			name: "Test_Store_GetCollectionSize_OK",
			args: args{
				entityMetadata: transaction.Provider().GetEntityMetadata(),
				collectionName: txn.GetCollectionName(),
			},
			want: -1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if got := ms.GetCollectionSize(tt.args.ctx, tt.args.entityMetadata, tt.args.collectionName); got != tt.want {
				t.Errorf("GetCollectionSize() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStore_AddToCollection(t *testing.T) {
	initDefaultTxnPool(t)
	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")

	type args struct {
		ctx context.Context
		ce  datastore.CollectionEntity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestStore_AddToCollection_Closed_Conn_ERR",
			args:    args{ce: &txn},
			wantErr: true,
		},
		{
			name:    "TestStore_AddToCollection_Closed_Conn_ERR",
			args:    args{ce: &txn},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.AddToCollection(tt.args.ctx, tt.args.ce); (err != nil) != tt.wantErr {
				t.Errorf("AddToCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTxn(t *testing.T) {
	type F struct {
		ClientID datastore.Key `msgpack:"cid"`
	}
	foo := F{
		ClientID: encryption.Hash("id"),
	}

	v, err := msgpack.Marshal(foo)
	require.NoError(t, err)

	var bar F
	err = msgpack.Unmarshal(v, &bar)
	require.NoError(t, err)
	fmt.Println(bar.ClientID)
	require.Equal(t, foo.ClientID, bar.ClientID)
}

func TestStore_DeleteFromCollection(t *testing.T) {
	initDefaultTxnPool(t)
	if err := memorystore.DefaultPool.Close(); err != nil {
		t.Fatal(err)
	}

	txn := transaction.Transaction{}
	txn.CollectionMemberField.EntityCollection = &datastore.EntityCollection{}
	txn.CollectionMemberField.EntityCollection.CollectionDuration = time.Millisecond
	txn.SetKey("key")

	type args struct {
		ctx context.Context
		ce  datastore.CollectionEntity
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestStore_DeleteFromCollection_Closed_Conn_ERR",
			args:    args{ce: &txn},
			wantErr: true,
		},
		{
			name:    "TestStore_DeleteFromCollection_Closed_Conn_ERR",
			args:    args{ce: &txn},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := &memorystore.Store{}
			if err := ms.DeleteFromCollection(tt.args.ctx, tt.args.ce); (err != nil) != tt.wantErr {
				t.Errorf("DeleteFromCollection() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
