package block

// Last events entity db stores the last N block events in the rocksdb.
// It will be used to retrieve the last N block events for kafka message syncing.

import (
	"context"
	"encoding/json"
	"path/filepath"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

type BlockEvents struct {
	datastore.NOIDField
	Key    string `json:"key"`
	Round  int64  `json:"round"`
	Events []byte `json:"events"`
}

var blockEventEntityMetadata *datastore.EntityMetadataImpl

// SetupBlockEventEntity - setup the block event entity
func SetupBlockEventEntity(store datastore.Store) {
	blockEventEntityMetadata = datastore.MetadataProvider()
	blockEventEntityMetadata.Name = "last_block_events"
	blockEventEntityMetadata.DB = "last_block_eventsdb"
	blockEventEntityMetadata.Provider = BlockEventProvider
	blockEventEntityMetadata.Store = store
	blockEventEntityMetadata.IDColumnName = "key"
	datastore.RegisterEntityMetadata("last_block_events", blockEventEntityMetadata)
}

// SetupBlockEventDB - sets up the last block events database
func SetupBlockEventDB(workdir string) {
	datadir := filepath.Join(workdir, "data/rocksdb/lastblockevents")
	db, err := ememorystore.CreateDB(datadir)
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool("last_block_eventsdb", db)
}

func BlockEventProvider() datastore.Entity {
	return &BlockEvents{}
}

/*GetEntityMetadata - implement interface */
func (b *BlockEvents) GetEntityMetadata() datastore.EntityMetadata {
	return blockEventEntityMetadata
}

/*GetKey - implement interface */
func (b *BlockEvents) GetKey() datastore.Key {
	return datastore.ToKey(b.Key)
}

/*SetKey - implement interface */
func (b *BlockEvents) SetKey(key datastore.Key) {
	b.Key = datastore.ToString(key)
}

/*Read - store read */
func (b *BlockEvents) Read(ctx context.Context, key datastore.Key) error {
	return b.GetEntityMetadata().GetStore().Read(ctx, key, b)
}

/*Write - store read */
func (b *BlockEvents) Write(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Write(ctx, b)
}

func (b *BlockEvents) MultiWrite(ctx context.Context, entities []datastore.Entity) error {
	return b.GetEntityMetadata().GetStore().MultiWrite(ctx, blockEventEntityMetadata, entities)
}

/*Delete - store read */
func (b *BlockEvents) Delete(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Delete(ctx, b)
}

func (b *BlockEvents) Encode() []byte {
	buff, _ := json.Marshal(b)
	return buff
}

func (b *BlockEvents) Decode(input []byte) error {
	return json.Unmarshal(input, b)
}
