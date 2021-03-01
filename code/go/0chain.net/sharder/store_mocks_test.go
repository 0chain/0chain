package sharder_test

import (
	"0chain.net/chaincore/round"
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

func (b2 blockStoreMock) Read(hash string, _ int64) (*block.Block, error) {
	v, ok := b2.blocks[hash]
	if !ok {
		return nil, errors.New("unknown block")
	}
	return v, nil
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
	blockSummaries map[string]*block.BlockSummary
	magicBlockMaps map[string]*block.MagicBlockMap
	rounds         map[string]*round.Round
}

var (
	_ datastore.Store = (*storeMock)(nil)
)

func NewStoreMock() storeMock {
	return storeMock{
		blockSummaries: make(map[string]*block.BlockSummary),
		magicBlockMaps: make(map[string]*block.MagicBlockMap),
		rounds:         make(map[string]*round.Round),
	}
}

func (s storeMock) Read(_ context.Context, key datastore.Key, entity datastore.Entity) error {
	if strings.Contains(key, "!") {
		return errors.New("key can not contain \"!\"")
	}

	name := entity.GetEntityMetadata().GetName()

	if (name == "block_summary" || name == "txn_summary") && len(key) != 64 {
		return errors.New("key length must be 64")
	}

	if name == "round" || name == "magic_block_map" {
		n, err := strconv.ParseInt(key, 10, 64)
		if err != nil {
			return err
		}
		if n < 0 {
			return errors.New("key can not be negative")
		}
	}

	switch name {
	case "block_summary":
		v, ok := s.blockSummaries[key]
		if !ok {
			return errors.New("unknown block summary")
		}

		bs := entity.(*block.BlockSummary)
		*bs = *v
	case "round":
		v, ok := s.rounds[key]
		if !ok {
			return errors.New("unknown round")
		}

		r := entity.(*round.Round)
		*r = *v
	case "magic_block_map":
		v, ok := s.magicBlockMaps[key]
		if !ok {
			return errors.New("unknown magic block map")
		}
		mb := entity.(*block.MagicBlockMap)
		*mb = *v

	}

	return nil
}

func (s storeMock) Write(ctx context.Context, entity datastore.Entity) error {
	name := entity.GetEntityMetadata().GetName()

	if (name == "block" || name == "block_summary") && len(entity.GetKey()) != 64 {
		return errors.New("key must be 64 size")
	}

	if name == "magic_block_map" || name == "round" {
		n, err := strconv.Atoi(entity.GetKey())
		if err != nil {
			return err
		}
		if n < 0 {
			return errors.New("key can not be negative")
		}
	}

	switch name {
	case "block_summary":
		bs := entity.(*block.BlockSummary)
		s.blockSummaries[entity.GetKey()] = bs
	case "magic_block_map":
		mb := entity.(*block.MagicBlockMap)
		s.magicBlockMaps[entity.GetKey()] = mb
	case "round":
		r := entity.(*round.Round)
		s.rounds[entity.GetKey()] = r
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
