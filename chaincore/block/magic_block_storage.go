package block

import (
	"context"
	"path/filepath"
	"strconv"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

type MagicBlockData struct {
	datastore.IDField
	*MagicBlock
}

var magicBlockMetadata *datastore.EntityMetadataImpl

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

func SetupMagicBlockDataDB(workdir string) {
	db, err := ememorystore.CreateDB(filepath.Join(workdir, "data/rocksdb/mb"))
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
