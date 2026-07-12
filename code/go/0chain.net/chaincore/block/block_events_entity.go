package block

// Block events entity db stores the last N block events in the rocksdb.
// It will be used to retrieve the last N block events for kafka message syncing.

import (
	"context"
	"encoding/json"
	"path/filepath"

	"0chain.net/core/datastore"
	"0chain.net/core/ememorystore"
)

const (
	BlockEventsMetaName = "block_events"
	BlockEventsDBName   = "block_eventsdb"
)

var (
	// EventsRingSize represents the length of the block events ring
	EventsRingSize           = 100
	blockEventEntityMetadata *datastore.EntityMetadataImpl
)

// BlockEvents represents the entity to store the block events in rocksdb
// The key is the round%BlockEventsRingSize.
type BlockEvents struct {
	datastore.NOIDField
	Key    string `json:"key"`
	Round  int64  `json:"round"`
	Events []byte `json:"events"`
}

// SetupBlockEventEntity - setup the block event entity
func SetupBlockEventEntity(store datastore.Store) {
	blockEventEntityMetadata = datastore.MetadataProvider()
	blockEventEntityMetadata.Name = BlockEventsMetaName
	blockEventEntityMetadata.DB = BlockEventsDBName
	blockEventEntityMetadata.Provider = BlockEventProvider
	blockEventEntityMetadata.Store = store
	blockEventEntityMetadata.IDColumnName = "key"
	datastore.RegisterEntityMetadata(BlockEventsMetaName, blockEventEntityMetadata)
}

// SetupBlockEventDB - sets up the last block events database
func SetupBlockEventDB(workdir string) {
	datadir := filepath.Join(workdir, "data/rocksdb/blockevents")
	db, err := ememorystore.CreateDB(datadir)
	if err != nil {
		panic(err)
	}
	ememorystore.AddPool(BlockEventsDBName, db)
}

func BlockEventProvider() datastore.Entity {
	return &BlockEvents{}
}

// GetEntityMetadata returns the blockEventEntityMetadata
func (b *BlockEvents) GetEntityMetadata() datastore.EntityMetadata {
	return blockEventEntityMetadata
}

// GetKey returns the key of the entity
func (b *BlockEvents) GetKey() datastore.Key {
	return datastore.ToKey(b.Key)
}

// SetKey sets the key of the entity
func (b *BlockEvents) SetKey(key datastore.Key) {
	b.Key = datastore.ToString(key)
}

// Read reads the block events from the store
func (b *BlockEvents) Read(ctx context.Context, key datastore.Key) error {
	return b.GetEntityMetadata().GetStore().Read(ctx, key, b)
}

// Write writes the block events to the store
func (b *BlockEvents) Write(ctx context.Context) error {
	return b.GetEntityMetadata().GetStore().Write(ctx, b)
}

// Delete deletes the block events from the store
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
