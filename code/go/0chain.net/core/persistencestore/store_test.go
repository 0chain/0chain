package persistencestore_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/0chain/common/core/logging"
	"github.com/gocql/gocql"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/mocks"
	"0chain.net/core/persistencestore"
)

func init() {
	block.SetupEntity(persistencestore.GetStorageProvider())
	logging.InitLogging("test", "")
}

func makeTestMocks() (*mocks.SessionI, *mocks.QueryI, *mocks.IteratorI) {
	var (
		sm = mocks.SessionI{}
		qm = mocks.QueryI{}
		im = mocks.IteratorI{}
	)

	sm.On("Query", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(
		func(_ string, _ ...interface{}) persistencestore.QueryI {
			return &qm
		},
	)

	qm.On("Exec").Return(
		func() error { return nil },
	)
	qm.On("Iter").Return(func() persistencestore.IteratorI {
		return &im
	})

	return &sm, &qm, &im
}

func TestStore_Read(t *testing.T) {
	sm, _, im := makeTestMocks()

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	persistencestore.Session = sm

	type args struct {
		ctx    context.Context
		key    datastore.Key
		entity datastore.Entity
	}
	tests := []struct {
		name        string
		args        args
		scan        bool
		wantIterErr bool
		wantErr     bool
	}{
		{
			name:        "Test_Store_Read_OK",
			args:        args{key: b.GetKey(), entity: block.Provider()},
			scan:        true,
			wantIterErr: false,
			wantErr:     false,
		},
		{
			name:    "Test_Store_Read_Not_Found_ERR",
			args:    args{key: b.GetKey(), entity: block.Provider()},
			scan:    false,
			wantErr: true,
		},
		{
			name:        "Test_Store_Read_Iter_ERR",
			args:        args{key: b.GetKey(), entity: block.Provider()},
			scan:        true,
			wantIterErr: true,
			wantErr:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im.On("Scan", mock.MatchedBy(func(s *string) bool {
				ps := &persistencestore.Store{}
				v, err := json.Marshal(ps)
				require.NoError(t, err)
				*s = string(v)
				return true
			})).Return(
				func(_ ...interface{}) bool {
					return tt.scan
				},
			)
			im.On("Close").Return(
				func() error {
					if tt.wantIterErr {
						return errors.New("")
					}
					return nil
				},
			)

			ps := &persistencestore.Store{}
			if err := ps.Read(tt.args.ctx, tt.args.key, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Read() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Write(t *testing.T) {
	sm, _, _ := makeTestMocks()

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	persistencestore.Session = sm

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
			name:    "Test_Store_Write_OK",
			args:    args{entity: b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &persistencestore.Store{}
			if err := ps.Write(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Write() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_InsertIfNE(t *testing.T) {
	sm, _, _ := makeTestMocks()

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	persistencestore.Session = sm

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
			args:    args{entity: b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &persistencestore.Store{}
			if err := ps.InsertIfNE(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("InsertIfNE() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_Delete(t *testing.T) {
	sm, _, _ := makeTestMocks()

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	persistencestore.Session = sm

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
			name:    "Test_Store_Delete_OK",
			args:    args{entity: b},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &persistencestore.Store{}
			if err := ps.Delete(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
				t.Errorf("Delete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiRead(t *testing.T) {
	sm, qm, im := makeTestMocks()

	sm.On("Query", mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("int"), mock.AnythingOfType("string")).Return(
		func() persistencestore.QueryI {
			return qm
		},
	)
	sm.On("Query", mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string"), mock.AnythingOfType("string"), mock.AnythingOfType("string"),
		mock.AnythingOfType("string")).Return(
		func(_ string, _ ...interface{}) persistencestore.QueryI {
			return qm
		},
	)

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	persistencestore.Session = sm

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		keys           []datastore.Key
		entities       []datastore.Entity
	}
	tests := []struct {
		name    string
		args    args
		scan    bool
		iterErr bool
		wantErr bool
	}{
		{
			name: "Test_Store_MultiRead_OK",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					b.GetKey(),
				},
				entities: []datastore.Entity{
					b,
				},
			},
			scan:    true,
			iterErr: false,
			wantErr: false,
		},
		{
			name: "Test_Store_MultiRead_Iter_ERR",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				keys: []datastore.Key{
					b.GetKey(),
				},
				entities: []datastore.Entity{
					b,
				},
			},
			scan:    false,
			iterErr: true,
			wantErr: true,
		},
		{
			name: "Test_Store_MultiRead_Size>BatchSize_OK",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, persistencestore.BATCH_SIZE+1),
				entities: func() []datastore.Entity {
					entities := make([]datastore.Entity, persistencestore.BATCH_SIZE+1)
					for ind := range entities {
						entities[ind] = block.Provider()
					}

					return entities
				}(),
			},
			scan:    true,
			iterErr: false,
			wantErr: false,
		},
		{
			name: "Test_Store_MultiRead_Size>BatchSize_ERR",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				keys:           make([]datastore.Key, persistencestore.BATCH_SIZE+1),
				entities: func() []datastore.Entity {
					entities := make([]datastore.Entity, persistencestore.BATCH_SIZE+1)
					for ind := range entities {
						entities[ind] = block.Provider()
					}

					return entities
				}(),
			},
			scan:    true,
			iterErr: true,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			im.On("Scan", mock.AnythingOfType("*string")).Return(
				func(_ ...interface{}) bool {
					return tt.scan
				},
			)

			im.On("Close").Return(
				func() error {
					if tt.iterErr {
						return errors.New("")
					}
					return nil
				},
			)

			ps := &persistencestore.Store{}
			if err := ps.MultiRead(tt.args.ctx, tt.args.entityMetadata, tt.args.keys, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("multiRead() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiWrite(t *testing.T) {
	sm, qm, _ := makeTestMocks()
	bm := mocks.BatchI{}

	sm.On("NewBatch", mock.AnythingOfType("gocql.BatchType")).Return(
		func(_ gocql.BatchType) persistencestore.BatchI {
			return &bm
		},
	)
	var a []interface{}
	sm.On("Query", mock.AnythingOfType("string"), a).Return(
		func(_ string, _ ...interface{}) persistencestore.QueryI {
			return qm
		},
	)
	bm.On("Query", mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(
		func(_ string, _ ...interface{}) {},
	)

	b := block.NewBlock("", 1)
	b.Hash = b.ComputeHash()

	persistencestore.Session = sm

	type args struct {
		ctx            context.Context
		entityMetadata datastore.EntityMetadata
		entities       []datastore.Entity
	}
	tests := []struct {
		name           string
		args           args
		simpleBatchErr bool
		wantErr        bool
	}{
		{
			name: "Test_Store_MultiWrite_OK",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					b,
				},
			},
			wantErr: false,
		},
		{
			name: "Test_Store_MultiWrite_ERR",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: []datastore.Entity{
					b,
				},
			},
			simpleBatchErr: true,
			wantErr:        true,
		},
		{
			name: "Test_Store_MultiWrite_Size>BatchSize_ERR",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: func() []datastore.Entity {
					entities := make([]datastore.Entity, persistencestore.BATCH_SIZE+1)
					for ind := range entities {
						entities[ind] = block.Provider()
					}

					return entities
				}(),
			},
			simpleBatchErr: true,
			wantErr:        true,
		},
		{
			name: "Test_Store_MultiWrite_Size>BatchSize_OK",
			args: args{
				entityMetadata: block.Provider().GetEntityMetadata(),
				entities: func() []datastore.Entity {
					entities := make([]datastore.Entity, persistencestore.BATCH_SIZE+1)
					for ind := range entities {
						entities[ind] = block.Provider()
					}

					return entities
				}(),
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ps := &persistencestore.Store{}
			sm.On("ExecuteBatch", mock.AnythingOfType("*mocks.BatchI")).Return(
				func(_ persistencestore.BatchI) error {
					if tt.simpleBatchErr {
						return errors.New("")
					}
					return nil
				},
			)

			if err := ps.MultiWrite(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiWrite() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStore_MultiDelete(t *testing.T) {
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
			name:    "Test_Store_MultiDelete_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &persistencestore.Store{}
			if err := ps.MultiDelete(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
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
			name:    "Test_Store_AddToCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &persistencestore.Store{}
			if err := ps.AddToCollection(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
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
			name:    "Test_Store_MultiAddToCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &persistencestore.Store{}
			if err := ps.MultiAddToCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
				t.Errorf("MultiAddToCollection() error = %v, wantErr %v", err, tt.wantErr)
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
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &persistencestore.Store{}
			if err := ps.IterateCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.collectionName, tt.args.handler); (err != nil) != tt.wantErr {
				t.Errorf("IterateCollection() error = %v, wantErr %v", err, tt.wantErr)
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
			name:    "Test_Store_DeleteFromCollection_OK",
			wantErr: false, // not implemented
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ps := &persistencestore.Store{}
			if err := ps.DeleteFromCollection(tt.args.ctx, tt.args.entity); (err != nil) != tt.wantErr {
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

			ps := &persistencestore.Store{}
			if err := ps.MultiDeleteFromCollection(tt.args.ctx, tt.args.entityMetadata, tt.args.entities); (err != nil) != tt.wantErr {
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

			ps := &persistencestore.Store{}
			if got := ps.GetCollectionSize(tt.args.ctx, tt.args.entityMetadata, tt.args.collectionName); got != tt.want {
				t.Errorf("GetCollectionSize() = %v, want %v", got, tt.want)
			}
		})
	}
}
