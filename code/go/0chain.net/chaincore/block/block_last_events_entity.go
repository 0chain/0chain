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

type LastBlockEvent struct {
	datastore.NOIDField
	Key      string `json:"key"`
	Sequence int64  `json:"sequence"`
	Round    int64  `json:"round"`
	Event    []byte `json:"event"`
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
	return &LastBlockEvent{}
}

/*GetEntityMetadata - implement interface */
func (b *LastBlockEvent) GetEntityMetadata() datastore.EntityMetadata {
	return blockEventEntityMetadata
}

/*GetKey - implement interface */
func (b *LastBlockEvent) GetKey() datastore.Key {
	return datastore.ToKey(b.Key)
}

/*SetKey - implement interface */
func (b *LastBlockEvent) SetKey(key datastore.Key) {
	b.Key = datastore.ToString(key)
}

/*Read - store read */
func (b *LastBlockEvent) Read(ctx context.Context, key datastore.Key) error {
	return b.GetEntityMetadata().GetStore().Read(ctx, key, b)
}

/*Write - store read */
func (b *LastBlockEvent) Write(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Write(ctx, b)
}

func (b *LastBlockEvent) MultiWrite(ctx context.Context, entities []datastore.Entity) error {
	return b.GetEntityMetadata().GetStore().MultiWrite(ctx, blockEventEntityMetadata, entities)
}

/*Delete - store read */
func (b *LastBlockEvent) Delete(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Delete(ctx, b)
}

func (b *LastBlockEvent) Encode() []byte {
	buff, _ := json.Marshal(b)
	return buff
}

func (b *LastBlockEvent) Decode(input []byte) error {
	return json.Unmarshal(input, b)
}
