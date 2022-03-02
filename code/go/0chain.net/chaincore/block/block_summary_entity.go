package block

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strconv"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
	"0chain.net/core/util"
)

/*BlockSummary - the summary of the block */
type BlockSummary struct {
	datastore.VersionField
	datastore.CreationDateField
	datastore.NOIDField

	Hash                  string        `json:"hash"`
	MinerID               datastore.Key `json:"miner_id"`
	Round                 int64         `json:"round"`
	RoundRandomSeed       int64         `json:"round_random_seed"`
	MerkleTreeRoot        string        `json:"merkle_tree_root"`
	ClientStateHash       util.Key      `json:"state_hash"`
	ReceiptMerkleTreeRoot string        `json:"receipt_merkle_tree_root"`
	NumTxns               int           `json:"num_txns"`
	*MagicBlock           `json:"maigc_block,omitempty" msgpack:"mb,omitempty"`
}

var blockSummaryEntityMetadata *datastore.EntityMetadataImpl

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

/*SetupBlockSummaryDB - sets up the block summary database */
func SetupBlockSummaryDB(workdir string) {
	datadir := filepath.Join(workdir, "data/rocksdb/blocksummary")
	db, err := ememorystore.CreateDB(datadir)
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("blocksummarydb", db)
}

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

func (b *BlockSummary) Encode() []byte {
	buff, _ := json.Marshal(b)
	return buff
}

func (b *BlockSummary) Decode(input []byte) error {
	return json.Unmarshal(input, b)
}

/*GetMagicBlockMap - get the magic block map of this block */
func (b *BlockSummary) GetMagicBlockMap() *MagicBlockMap {
	if b.MagicBlock != nil {
		mbm := datastore.GetEntityMetadata("magic_block_map").Instance().(*MagicBlockMap)
		mbm.ID = strconv.FormatInt(b.MagicBlockNumber, 10)
		mbm.Hash = b.Hash
		mbm.BlockRound = b.Round
		return mbm
	}
	return nil
}
