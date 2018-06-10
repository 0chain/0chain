package persistencestore

import (
	"context"
	"testing"

	"0chain.net/block"
	"0chain.net/common"
	"0chain.net/datastore"
)

func TestInsert(t *testing.T) {
	b := block.BlockSummaryProvider().(*block.BlockSummary)
	b.Hash = "abc"
	b.MerkleRoot = "def"
	b.Round = 0
	b.CreationDate = common.Now()
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
	b := &block.BlockSummary{}
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
	b := block.BlockSummaryProvider().(*block.BlockSummary)
	b.Hash = "abc"
	b.MerkleRoot = "def"
	b.Round = 1
	b.CreationDate = common.Now()
	ctx := context.Background()
	ctx = WithEntityConnection(ctx, b.GetEntityMetadata())
	store := Store{}
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
	store := Store{}
	err := store.Delete(ctx, b)
	if err != nil {
		t.Errorf("Error deleting the entity: %v\n", err.Error())
	} else {
		t.Logf("successfully deleted the entity: %v\n", b.GetKey())
	}
}
