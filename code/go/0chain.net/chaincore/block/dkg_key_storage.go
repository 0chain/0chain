package block

import (
	"context"
	"path/filepath"
	"strconv"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

type DKGKey struct {
	MagicBlockNum int64
	KeyShare      string
}

type DKGKeyData struct {
	datastore.IDField
	DKGKey *DKGKey
}

var dkgKeyMetadata *datastore.EntityMetadataImpl

func (dkgKey *DKGKeyData) GetEntityMetadata() datastore.EntityMetadata {
	return dkgKeyMetadata
}

func DKGKeyProvider() datastore.Entity {
	dkgKey := &DKGKeyData{}
	return dkgKey
}

func SetupDKGKeyEntity(store datastore.Store) {
	dkgKeyMetadata = datastore.MetadataProvider()
	dkgKeyMetadata.Name = "dkgkeydata"
	dkgKeyMetadata.DB = "dkgkeydb"
	dkgKeyMetadata.Store = store
	dkgKeyMetadata.Provider = DKGKeyProvider
	dkgKeyMetadata.IDColumnName = "id"
	datastore.RegisterEntityMetadata("dkgkeydata", dkgKeyMetadata)
}

func SetupDKGKeyDB(workdir string) {
	db, err := ememorystore.CreateDB(filepath.Join(workdir, "data/rocksdb/dkgkey"))
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("dkgkeydb", db)
}

func (dkgKey *DKGKeyData) Read(ctx context.Context, key string) error {
	return dkgKey.GetEntityMetadata().GetStore().Read(ctx, key, dkgKey)
}

func (dkgKey *DKGKeyData) Write(ctx context.Context) error {
	return dkgKey.GetEntityMetadata().GetStore().Write(ctx, dkgKey)
}

func (dkgKey *DKGKeyData) Delete(ctx context.Context) error {
	return dkgKey.GetEntityMetadata().GetStore().Delete(ctx, dkgKey)
}

func NewDKGKeyData(dkgKey *DKGKey) *DKGKeyData {
	dkgKeyData := datastore.GetEntityMetadata("dkgkeydata").Instance().(*DKGKeyData)
	dkgKeyData.ID = strconv.FormatInt(dkgKey.MagicBlockNum, 10)
	dkgKeyData.DKGKey = dkgKey
	return dkgKeyData
}
