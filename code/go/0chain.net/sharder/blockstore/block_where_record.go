package blockstore

import (
	"context"

	"0chain.net/core/datastore"
)

// blockWhereRecord It simply provides whereabouts of a block. It can be in Warm Tier, Cold Tier, Hot and Warm Tier, Hot and Cold Tier, etc.
type blockWhereRecord struct {
	Hash      string    `json:"-"`
	Tiering   WhichTier `json:"tr"`
	BlockPath string    `json:"vp,omitempty"` // For disk volume it is simple unix path. For cold storage it is "storageUrl:bucketName".
	ColdPath  string    `json:"cp,omitempty"`
}

func NewBlockWhereRecord(hash string, tiering WhichTier, blockPath, coldPath string) *blockWhereRecord {
	bwr := &blockWhereRecord{Hash: hash, Tiering: tiering, BlockPath: blockPath, ColdPath: coldPath}
	return bwr
}

func DefaultBlockWhereRecord() *blockWhereRecord {
	return &blockWhereRecord{}
}

func (bwr *blockWhereRecord) GetEntityMetadata() datastore.EntityMetadata {
	return blockWhereRecordEntityMetadata
}

func (bwr *blockWhereRecord) SetKey(key datastore.Key) {
	bwr.Hash = datastore.ToString(key)
}

func (bwr *blockWhereRecord) GetKey() datastore.Key {
	return datastore.ToKey(bwr.Hash)
}

func (bwr *blockWhereRecord) GetScore() int64 {
	return 0 // Not implemented
}

func (bwr *blockWhereRecord) ComputeProperties() {
	// Not implemented
}

func (bwr *blockWhereRecord) Validate(ctx context.Context) error {
	return nil // Not implemented
}

func (bwr *blockWhereRecord) Read(ctx context.Context, key datastore.Key) error {
	return bwr.GetEntityMetadata().GetStore().Read(ctx, key, bwr)
}

func (bwr *blockWhereRecord) Write(ctx context.Context) error {
	return bwr.GetEntityMetadata().GetStore().Write(ctx, bwr)
}

func (bwr *blockWhereRecord) Delete(ctx context.Context) error {
	return bwr.GetEntityMetadata().GetStore().Delete(ctx, bwr)
}

var blockWhereRecordEntityMetadata *datastore.EntityMetadataImpl

// providerBlockWhereRecord - entity provider for client object
func providerBlockWhereRecord() datastore.Entity {
	return &blockWhereRecord{}
}

// setupEntityBlockWhereRecord - setup the entity
func setupEntityBlockWhereRecord(store datastore.Store) {
	blockWhereRecordEntityMetadata = datastore.MetadataProvider()
	blockWhereRecordEntityMetadata.Name = "bwr"
	blockWhereRecordEntityMetadata.DB = "MetaRecordDB"
	blockWhereRecordEntityMetadata.Provider = providerBlockWhereRecord
	blockWhereRecordEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("bwr", blockWhereRecordEntityMetadata)
}
