package block

import (
	"context"

	"0chain.net/common"
	"0chain.net/datastore"
	"0chain.net/ememorystore"
	"0chain.net/util"
)

/*BlockSummary - the summary of the block */
type BlockSummary struct {
	datastore.VersionField
	datastore.CreationDateField
	datastore.NOIDField
	Hash                  string   `json:"hash"`
	Round                 int64    `json:"round"`
	RoundRandomSeed       int64    `json:"round_random_seed"`
	MerkleTreeRoot        string   `json:"merkle_tree_root"`
	ClientStateHash       util.Key `json:"state_hash"`
	ReceiptMerkleTreeRoot string   `json:"receipt_merkle_tree_root"`
}

/*SetupBlockSummaryDB - sets up the block summary database */
func SetupBlockSummaryDB() {
	db, err := ememorystore.CreateDB("data/rocksdb/blocksummary")
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("blocksummarydb", db)
}

var blockSummaryEntityMetadata *datastore.EntityMetadataImpl

/*BlockSummaryProvider - a block summary instance provider */
func BlockSummaryProvider() datastore.Entity {
	b := &BlockSummary{}
	b.Version = "1.0"
	b.CreationDate = common.Now()
	return b
}

/*GetEntityMetadata - implement interface */
func (b *BlockSummary) GetEntityMetadata() datastore.EntityMetadata {
	return blockSummaryEntityMetadata
}

/*GetKey - implement interface */
func (b *BlockSummary) GetKey() datastore.Key {
	return datastore.ToKey(b.Hash)
}

/*SetKey - implement interface */
func (b *BlockSummary) SetKey(key datastore.Key) {
	b.Hash = datastore.ToString(key)
}

/*Read - store read */
func (b *BlockSummary) Read(ctx context.Context, key datastore.Key) error {
	return b.GetEntityMetadata().GetStore().Read(ctx, key, b)
}

/*Write - store read */
func (b *BlockSummary) Write(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Write(ctx, b)
}

/*Delete - store read */
func (b *BlockSummary) Delete(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Delete(ctx, b)
}

/*SetupBlockSummaryEntity - setup the block summary entity */
func SetupBlockSummaryEntity(store datastore.Store) {
	blockSummaryEntityMetadata = datastore.MetadataProvider()
	blockSummaryEntityMetadata.Name = "block_summary"
	blockSummaryEntityMetadata.DB = "blocksummarydb"
	blockSummaryEntityMetadata.Provider = BlockSummaryProvider
	blockSummaryEntityMetadata.Store = store
	blockSummaryEntityMetadata.IDColumnName = "hash"
	datastore.RegisterEntityMetadata("block_summary", blockSummaryEntityMetadata)
}
