package datastore_test

import (
	"reflect"
	"testing"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/core/memorystore"
)

func makeTestEntityMetadataImpl() datastore.EntityMetadataImpl {
	block.SetupEntity(memorystore.GetStorageProvider())
	em := datastore.GetEntityMetadata("block")

	return datastore.EntityMetadataImpl{
		Name:         em.GetName(),
		DB:           em.GetDB(),
		Store:        em.GetStore(),
		Provider:     em.Instance,
		IDColumnName: em.GetIDColumnName(),
	}
}

func TestEntityMetadataImpl_GetDB(t *testing.T) {
	t.Parallel()

	em := makeTestEntityMetadataImpl()

	type fields struct {
		Name         string
		DB           string
		Store        datastore.Store
		Provider     datastore.InstanceProvider
		IDColumnName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_EntityMetadataImpl_GetDB_OK",
			fields: fields{
				Name:         em.Name,
				DB:           em.DB,
				Store:        em.Store,
				Provider:     em.Provider,
				IDColumnName: em.IDColumnName,
			},
			want: em.DB,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			em := &datastore.EntityMetadataImpl{
				Name:         tt.fields.Name,
				DB:           tt.fields.DB,
				Store:        tt.fields.Store,
				Provider:     tt.fields.Provider,
				IDColumnName: tt.fields.IDColumnName,
			}
			if got := em.GetDB(); got != tt.want {
				t.Errorf("GetDB() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityMetadataImpl_GetIDColumnName(t *testing.T) {
	t.Parallel()

	em := makeTestEntityMetadataImpl()
	type fields struct {
		Name         string
		DB           string
		Store        datastore.Store
		Provider     datastore.InstanceProvider
		IDColumnName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_EntityMetadataImpl_GetIDColumnName_OK",
			fields: fields{
				Name:         em.Name,
				DB:           em.DB,
				Store:        em.Store,
				Provider:     em.Provider,
				IDColumnName: em.IDColumnName,
			},
			want: em.IDColumnName,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			em := &datastore.EntityMetadataImpl{
				Name:         tt.fields.Name,
				DB:           tt.fields.DB,
				Store:        tt.fields.Store,
				Provider:     tt.fields.Provider,
				IDColumnName: tt.fields.IDColumnName,
			}
			if got := em.GetIDColumnName(); got != tt.want {
				t.Errorf("GetIDColumnName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityMetadataImpl_GetName(t *testing.T) {
	t.Parallel()

	em := makeTestEntityMetadataImpl()

	type fields struct {
		Name         string
		DB           string
		Store        datastore.Store
		Provider     datastore.InstanceProvider
		IDColumnName string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Test_EntityMetadataImpl_GetName_OK",
			fields: fields{
				Name:         em.Name,
				DB:           em.DB,
				Store:        em.Store,
				Provider:     em.Provider,
				IDColumnName: em.IDColumnName,
			},
			want: em.Name,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			em := &datastore.EntityMetadataImpl{
				Name:         tt.fields.Name,
				DB:           tt.fields.DB,
				Store:        tt.fields.Store,
				Provider:     tt.fields.Provider,
				IDColumnName: tt.fields.IDColumnName,
			}
			if got := em.GetName(); got != tt.want {
				t.Errorf("GetName() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityMetadataImpl_GetStore(t *testing.T) {
	t.Parallel()

	em := makeTestEntityMetadataImpl()

	type fields struct {
		Name         string
		DB           string
		Store        datastore.Store
		Provider     datastore.InstanceProvider
		IDColumnName string
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Store
	}{
		{
			name: "Test_EntityMetadataImpl_GetName_OK",
			fields: fields{
				Name:         em.Name,
				DB:           em.DB,
				Store:        em.Store,
				Provider:     em.Provider,
				IDColumnName: em.IDColumnName,
			},
			want: em.Store,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			em := &datastore.EntityMetadataImpl{
				Name:         tt.fields.Name,
				DB:           tt.fields.DB,
				Store:        tt.fields.Store,
				Provider:     tt.fields.Provider,
				IDColumnName: tt.fields.IDColumnName,
			}
			if got := em.GetStore(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetStore() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEntityMetadataImpl_Instance(t *testing.T) {
	t.Parallel()

	em := makeTestEntityMetadataImpl()

	type fields struct {
		Name         string
		DB           string
		Store        datastore.Store
		Provider     datastore.InstanceProvider
		IDColumnName string
	}
	tests := []struct {
		name   string
		fields fields
		want   datastore.Entity
	}{
		{
			name: "Test_EntityMetadataImpl_Instance_OK",
			fields: fields{
				Name:         em.Name,
				DB:           em.DB,
				Store:        em.Store,
				Provider:     em.Provider,
				IDColumnName: em.IDColumnName,
			},
			want: em.Provider(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			em := &datastore.EntityMetadataImpl{
				Name:         tt.fields.Name,
				DB:           tt.fields.DB,
				Store:        tt.fields.Store,
				Provider:     tt.fields.Provider,
				IDColumnName: tt.fields.IDColumnName,
			}
			if got := em.Instance(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Instance() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEntity(t *testing.T) {
	t.Parallel()

	blockEntityMetadata := datastore.MetadataProvider()
	blockEntityMetadata.Name = "block"
	blockEntityMetadata.Provider = block.Provider
	blockEntityMetadata.Store = memorystore.GetStorageProvider()
	blockEntityMetadata.IDColumnName = "hash"

	type args struct {
		entityName string
	}
	tests := []struct {
		name string
		args args
		want datastore.Entity
	}{
		{
			name: "Test_GetEntity_OK",
			args: args{entityName: "block"},
			want: blockEntityMetadata.Instance(),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.GetEntity(tt.args.entityName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetEntity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetEntityMetadata(t *testing.T) {
	t.Parallel()

	blockEntityMetadata := datastore.MetadataProvider()
	blockEntityMetadata.Name = "block"
	blockEntityMetadata.Provider = block.Provider
	blockEntityMetadata.Store = memorystore.GetStorageProvider()
	blockEntityMetadata.IDColumnName = "hash"

	type args struct {
		entityName string
	}
	tests := []struct {
		name string
		args args
		want datastore.EntityMetadata
	}{
		{
			name: "Test_GetEntityMetadata_OK",
			args: args{entityName: "block"},
			want: blockEntityMetadata,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := datastore.GetEntityMetadata(tt.args.entityName); !reflect.DeepEqual(got.Instance(), tt.want.Instance()) {
				t.Errorf("GetEntityMetadata() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
