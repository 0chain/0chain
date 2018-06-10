package block

import (
	"0chain.net/common"
	"0chain.net/datastore"
)

type BlockSummary struct {
	datastore.VersionField
	datastore.CreationDateField
	datastore.NOIDField
	Hash       string `json:"hash"`
	MerkleRoot string `json:"merkle_root"`
	Round      int64  `json:"round"`
}

var blockSummaryEntityMetadata *datastore.EntityMetadataImpl

func BlockSummaryProvider() datastore.Entity {
	b := &BlockSummary{}
	b.Version = "1.0"
	b.CreationDate = common.Now()
	return b
}

func (b *BlockSummary) GetEntityName() string {
	return "block_summary"
}

func (b *BlockSummary) GetEntityMetadata() datastore.EntityMetadata {
	return blockSummaryEntityMetadata
}

func (b *BlockSummary) GetKey() datastore.Key {
	return datastore.ToKey(b.Hash)
}

func (b *BlockSummary) SetKey(key datastore.Key) {
	b.Hash = datastore.ToString(key)
}

func SetupBlockSummaryEntity(store datastore.Store) {
	blockSummaryEntityMetadata = datastore.MetadataProvider()
	blockSummaryEntityMetadata.Name = "block_summary"
	blockSummaryEntityMetadata.Provider = BlockSummaryProvider
	blockSummaryEntityMetadata.Store = store
	blockSummaryEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("block_summary", blockSummaryEntityMetadata)
}
