package sharder_test

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"0chain.net/chaincore/block"
	"0chain.net/core/datastore"
	"0chain.net/sharder/blockstore"
)

type blockStoreMock struct {
	cloud  map[string]struct{} // map to store cloud objects
	blocks map[string]*block.Block
}

var (
	_ blockstore.BlockStore = (*blockStoreMock)(nil)
)

func (b2 blockStoreMock) Write(b *block.Block) error {
	if len(b.Hash) != 64 {
		return errors.New("hash must be 64 size")
	}
	b2.blocks[b.Hash] = b
	return nil
}

func (b2 blockStoreMock) Read(_ string, _ int64) (*block.Block, error) {
	return nil, nil
}

func (b2 blockStoreMock) ReadWithBlockSummary(bs *block.BlockSummary) (*block.Block, error) {
	v, ok := b2.blocks[bs.Hash]
	if !ok {
		return nil, errors.New("unknown block")
	}
	return v, nil
}

func (b2 blockStoreMock) Delete(hash string) error {
	if len(hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	return nil
}

func (b2 blockStoreMock) DeleteBlock(b *block.Block) error {
	if len(b.Hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	return nil
}

func (b2 blockStoreMock) UploadToCloud(hash string, _ int64) error {
	if len(hash) != 64 {
		return errors.New("hash must be 64 size")
	}

	b2.cloud[hash] = struct{}{}
	return nil
}

func (b2 blockStoreMock) DownloadFromCloud(_ string, _ int64) error {
	return nil
}

func (b2 blockStoreMock) CloudObjectExists(hash string) bool {
	if len(hash) != 64 {
		return false
	}
	_, ok := b2.cloud[hash]
	return ok
}

type storeMock struct {
	blockSummaries map[string]block.BlockSummary
}

var (
	_ datastore.Store = (*storeMock)(nil)
)

func (s storeMock) Read(_ context.Context, key datastore.Key, entity datastore.Entity) error {
	if strings.Contains(key, "!") {
		return errors.New("key can not contain \"!\"")
	}

	name := entity.GetEntityMetadata().GetName()

	if name == "block_summary" && len(key) != 64 {
		return errors.New("block summaries hash length must be 64")
	}

	if name == "round" {
		n, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return err
		}
		if n < 0 {
			return errors.New("round key can not be negative")
		}
	}

	return nil
}

func (s storeMock) Write(ctx context.Context, entity datastore.Entity) error {
	name := entity.GetEntityMetadata().GetName()
	if len(entity.GetKey()) != 64 && (name == "block" || name == "block_summary") {
		return errors.New("key must be 64 size")
	}

	if name == "round" {
		num, err := strconv.Atoi(entity.GetKey())
		if err != nil {
			return err
		}

		if num < 0 {
			return errors.New("round num must be positive")
		}
	}

	return nil
}

func (s storeMock) InsertIfNE(ctx context.Context, entity datastore.Entity) error { return nil }

func (s storeMock) Delete(ctx context.Context, entity datastore.Entity) error { return nil }

func (s storeMock) MultiRead(ctx context.Context, entityMetadata datastore.EntityMetadata, keys []datastore.Key, entities []datastore.Entity) error {
	return nil
}

func (s storeMock) MultiWrite(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (s storeMock) MultiDelete(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (s storeMock) AddToCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	return nil
}

func (s storeMock) MultiAddToCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (s storeMock) DeleteFromCollection(ctx context.Context, entity datastore.CollectionEntity) error {
	return nil
}

func (s storeMock) MultiDeleteFromCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, entities []datastore.Entity) error {
	return nil
}

func (s storeMock) GetCollectionSize(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string) int64 {
	return 0
}

func (s storeMock) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	return nil
}
