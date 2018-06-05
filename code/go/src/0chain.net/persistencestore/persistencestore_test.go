package persistencestore

import (
	"context"
	"testing"
	"time"

	"0chain.net/datastore"
)

type Block struct {
	datastore.NOIDField
	Hash       string `json:"hash"`
	MerkleRoot string `json:"merkle_root"`
	Round      int64  `json:"round"`
	Timestamp  int64  `json:"timestamp"`
	NumTxns    int    `json:"num_txns"`
}

var blockEntityMetadata = datastore.MetadataProvider()

func init() {
	blockEntityMetadata.Name = "block"
	blockEntityMetadata.Provider = Provider
	blockEntityMetadata.IDColumnName = "hash"
}

func Provider() datastore.Entity {
	b := &Block{}
	b.Timestamp = time.Now().Unix()
	return b
}

func (b *Block) GetEntityName() string {
	return "block"
}

func (b *Block) GetEntityMetadata() datastore.EntityMetadata {
	return blockEntityMetadata
}

func (b *Block) GetKey() datastore.Key {
	return datastore.ToKey(b.Hash)
}

func TestInsert(t *testing.T) {
	b := &Block{Hash: "abc", MerkleRoot: "def", Round: 0, Timestamp: time.Now().Unix(), NumTxns: 5000}
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := Store{}
	err := store.Write(ctx, b)
	if err != nil {
		t.Errorf("Error writing the entity: %v\n", err.Error())
	}
}

func TestRead(t *testing.T) {
	key := "abc"
	b := &Block{}
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := Store{}
	err := store.Read(ctx, key, b)
	if err != nil {
		t.Errorf("Error reading the entity: %v\n", err.Error())
	} else {
		t.Logf("Entity: %v\n", datastore.ToJSON(b))
	}
}

func TestInsertIfNE(t *testing.T) {
	b := &Block{Hash: "abc", MerkleRoot: "def", Round: 0, Timestamp: time.Now().Unix(), NumTxns: 10000}
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := Store{}
	err := store.InsertIfNE(ctx, b)
	if err != nil {
		t.Errorf("Error inserting the entity: %v\n", err.Error())
	} else {
		t.Logf("should not have num txns as: %v\n", b.NumTxns)
	}
}

func TestDelete(t *testing.T) {
	b := &Block{Hash: "abc", MerkleRoot: "def", Round: 0, Timestamp: time.Now().Unix(), NumTxns: 10000}
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := Store{}
	err := store.Delete(ctx, b)
	if err != nil {
		t.Errorf("Error deleting the entity: %v\n", err.Error())
	} else {
		t.Logf("successfully deleted the entity: %v\n", b.GetKey())
	}
}
