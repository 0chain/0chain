package block

import (
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"context"
	"strconv"
)

type MagicBlockData struct {
	datastore.IDField
	*MagicBlock
}

var (
	magicBlockMetadata         *datastore.EntityMetadataImpl
	latestMagicBlockIDMetadata *datastore.EntityMetadataImpl
)

func (m *MagicBlockData) GetEntityMetadata() datastore.EntityMetadata {
	return magicBlockMetadata
}

func MagicBlockDataProvider() datastore.Entity {
	return &MagicBlockData{}
}

func SetupMagicBlockData(store datastore.Store) {
	magicBlockMetadata = datastore.MetadataProvider()
	magicBlockMetadata.Name = "magicblockdata"
	magicBlockMetadata.DB = "magicblockdatadb"
	magicBlockMetadata.Store = store
	magicBlockMetadata.Provider = MagicBlockDataProvider
	datastore.RegisterEntityMetadata("magicblockdata", magicBlockMetadata)
}

func SetupMagicBlockDataDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/mb")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("magicblockdatadb", db)
}

func (m *MagicBlockData) Read(ctx context.Context, key string) error {
	return m.GetEntityMetadata().GetStore().Read(ctx, key, m)
}

func (m *MagicBlockData) Write(ctx context.Context) error {
	return m.GetEntityMetadata().GetStore().Write(ctx, m)
}

func (m *MagicBlockData) Delete(ctx context.Context) error {
	return m.GetEntityMetadata().GetStore().Delete(ctx, m)
}

func NewMagicBlockData(mb *MagicBlock) *MagicBlockData {
	mbData := datastore.GetEntityMetadata("magicblockdata").Instance().(*MagicBlockData)
	mbData.ID = strconv.FormatInt(mb.MagicBlockNumber, 10)
	mbData.MagicBlock = mb
	return mbData
}

//
// Latest magic block ID storage.
//

func (lmbid *LatestMagicBlockID) GetEntityMetadata() datastore.EntityMetadata {
	return latestMagicBlockIDMetadata
}

func LatestMagicBlockIDProvider() datastore.Entity {
	return new(LatestMagicBlockID)
}

func SetupLatestMagicBlockID(store datastore.Store) {
	latestMagicBlockIDMetadata = datastore.MetadataProvider()
	latestMagicBlockIDMetadata.Name = "latestmbid"
	latestMagicBlockIDMetadata.DB = "latestmbiddb"
	latestMagicBlockIDMetadata.Store = store
	latestMagicBlockIDMetadata.Provider = LatestMagicBlockIDProvider
	datastore.RegisterEntityMetadata("latestmbid", latestMagicBlockIDMetadata)
}

func SetupLatestMagicBlockIDDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/latestmbid")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("latestmbiddb", db)
}

func (lmbid *LatestMagicBlockID) Read(ctx context.Context, key string) error {
	return lmbid.GetEntityMetadata().GetStore().Read(ctx, key, lmbid)
}

func (lmbid *LatestMagicBlockID) Write(ctx context.Context) error {
	return lmbid.GetEntityMetadata().GetStore().Write(ctx, lmbid)
}

func (lmbid *LatestMagicBlockID) Delete(ctx context.Context) error {
	return lmbid.GetEntityMetadata().GetStore().Delete(ctx, lmbid)
}
