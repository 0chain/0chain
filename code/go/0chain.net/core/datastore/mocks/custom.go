package mocks

import (
	"0chain.net/chaincore/block"
	"0chain.net/chaincore/round"
	"0chain.net/core/datastore"
	"context"
	"errors"
	"strconv"
)

type StoreCustom struct {
	blockSummaries map[string]*block.BlockSummary
	magicBlockMaps map[string]*block.MagicBlockMap
	rounds         map[string]*round.Round
}

var (
	_ datastore.Store = (*StoreCustom)(nil)
)

func NewStoreMock() StoreCustom {
	return StoreCustom{
		blockSummaries: make(map[string]*block.BlockSummary),
		magicBlockMaps: make(map[string]*block.MagicBlockMap),
		rounds:         make(map[string]*round.Round),
	}
}

func (s StoreCustom) Read(_ context.Context, key datastore.Key, entity datastore.Entity) error {
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

func (s StoreCustom) Write(_ context.Context, entity datastore.Entity) error {
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

func (s StoreCustom) InsertIfNE(_ context.Context, _ datastore.Entity) error {
	return nil
}

func (s StoreCustom) Delete(_ context.Context, _ datastore.Entity) error {
	return nil
}

func (s StoreCustom) MultiRead(_ context.Context, _ datastore.EntityMetadata, _ []datastore.Key, _ []datastore.Entity) error {
	return nil
}

func (s StoreCustom) MultiWrite(_ context.Context, _ datastore.EntityMetadata, _ []datastore.Entity) error {
	return nil
}

func (s StoreCustom) MultiDelete(_ context.Context, _ datastore.EntityMetadata, _ []datastore.Entity) error {
	return nil
}

func (s StoreCustom) AddToCollection(_ context.Context, _ datastore.CollectionEntity) error {
	return nil
}

func (s StoreCustom) MultiAddToCollection(_ context.Context, _ datastore.EntityMetadata, _ []datastore.Entity) error {
	return nil
}

func (s StoreCustom) DeleteFromCollection(_ context.Context, _ datastore.CollectionEntity) error {
	return nil
}

func (s StoreCustom) MultiDeleteFromCollection(_ context.Context, _ datastore.EntityMetadata, _ []datastore.Entity) error {
	return nil
}

func (s StoreCustom) GetCollectionSize(_ context.Context, _ datastore.EntityMetadata, _ string) int64 {
	return 0
}

func (s StoreCustom) IterateCollection(_ context.Context, _ datastore.EntityMetadata, _ string, _ datastore.CollectionIteratorHandler) error {
	return nil
}
