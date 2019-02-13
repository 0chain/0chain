package persistencestore

import (
	"context"
	"fmt"
	"testing"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

func init() {
	block.SetupBlockSummaryEntity(GetStorageProvider())
}

func TestInsert(t *testing.T) {
	b := block.BlockSummaryProvider().(*block.BlockSummary)
	b.Hash = "abcd"
	b.MerkleRoot = "defd"
	b.Round = 1
	b.CreationDate = common.Now()
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := GetStorageProvider()
	err := store.Write(ctx, b)
	if err != nil {
		t.Errorf("Error writing the entity: %v\n", err.Error())
	}
}

func TestRead(t *testing.T) {
	key := "abc"
	b := block.BlockSummaryProvider().(*block.BlockSummary)
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := GetStorageProvider()
	err := store.Read(ctx, key, b)
	if err != nil {
		t.Errorf("Error reading the entity: %v\n", err.Error())
	} else {
		t.Logf("Entity: %v\n", datastore.ToJSON(b))
	}
}

func TestInsertIfNE(t *testing.T) {
	b := block.BlockSummaryProvider().(*block.BlockSummary)
	b.Hash = "abc"
	b.MerkleRoot = "def"
	b.Round = 1
	b.CreationDate = common.Now()
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := GetStorageProvider()
	err := store.InsertIfNE(ctx, b)
	if err != nil {
		t.Errorf("Error inserting the entity: %v\n", err.Error())
	} else {
		t.Logf("should not have num txns as: %v\n", b.Round)
	}
}

func TestDelete(t *testing.T) {
	b := block.BlockSummaryProvider().(*block.BlockSummary)
	b.Hash = "abc"
	b.MerkleRoot = "def"
	b.Round = 0
	b.CreationDate = common.Now()
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := GetStorageProvider()
	err := store.Delete(ctx, b)
	if err != nil {
		t.Errorf("Error deleting the entity: %v\n", err.Error())
	} else {
		t.Logf("successfully deleted the entity: %v\n", b.GetKey())
	}
}

func TestMultiWrite(t *testing.T) {
	blockEntityMetadata := datastore.GetEntityMetadata("block_summary")
	blocks := datastore.AllocateEntities(1000, blockEntityMetadata)
	for idx, blk := range blocks {
		b := blk.(*block.BlockSummary)
		b.Hash = fmt.Sprintf("test_multi_insert_%v", idx)
		b.Round = int64(idx)
	}
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, blockEntityMetadata)
	store := GetStorageProvider()
	err := store.MultiWrite(ctx, blockEntityMetadata, blocks)
	if err != nil {
		t.Errorf("Error reading the entity: %v\n", err.Error())
	}
}

func TestMultiRead(t *testing.T) {
	blockEntityMetadata := datastore.GetEntityMetadata("block_summary")
	blocks := datastore.AllocateEntities(BATCH_SIZE, blockEntityMetadata)
	ctx := context.Background()
	start := time.Now()
	ctx = WithEntityConnection(ctx, blockEntityMetadata)
	store := GetStorageProvider()
	fmt.Printf("debug : %v\n", time.Since(start))
	keys := make([]datastore.Key, len(blocks))
	for idx := range keys {
		keys[idx] = datastore.ToKey(fmt.Sprintf("test_multi_insert_%v", idx))
	}
	err := store.MultiRead(ctx, blockEntityMetadata, keys, blocks)
	if err != nil {
		t.Errorf("Error reading the entity: %v\n", err.Error())
	} else {
		for idx, key := range keys {
			if key != blocks[idx].GetKey() {
				t.Logf("Entity: %v %v %v\n", idx, key, datastore.ToJSON(blocks[idx]))
			}
		}
	}
}
