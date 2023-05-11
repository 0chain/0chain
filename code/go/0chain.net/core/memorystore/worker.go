package memorystore

import (
	"context"
	"time"

	"0chain.net/core/datastore"
	. "github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

/*SetupWorkers - This setups up workers that allows aggregating and storing entities in chunks */
func SetupWorkers(ctx context.Context, options *datastore.ChunkingOptions) chan datastore.QueuedEntity {
	echannel := make(chan datastore.QueuedEntity, options.EntityBufferSize)
	bchannel := make(chan datastore.Chunk, options.ChunkBufferSize)
	var ecb datastore.EntityChunkBuilder
	ecb.ChunkProvider = &MemoryDBChunkProvider{}
	ecb.ChunkSize = options.ChunkSize
	ecb.MaxHoldupTime = options.MaxHoldupTime
	ecb.EntityChannel = echannel
	ecb.ChunkChannel = bchannel
	ecb.TimeoutChannel = time.NewTimer(-100 * time.Second)
	bworkers := make([]datastore.ChunkProcessor, options.NumChunkStorers)
	for i := 0; i < len(bworkers); i++ {
		bworker := &MemoryDBChunkProcessor{}
		bworker.EntityMetadata = options.EntityMetadata
		bworker.ChunkChannel = bchannel
		bworkers[i] = bworker
	}
	datastore.SetupChunkWorkers(ctx, ecb, bworkers)
	return echannel
}

/*MemoryDBChunk - implement db chunk */
type MemoryDBChunk struct {
	Buffer []datastore.Entity
	Length int
}

/*Size - interface implementation */
func (mc *MemoryDBChunk) Size() int {
	return mc.Length
}

/*Add - interface implementation */
func (mc *MemoryDBChunk) Add(entity datastore.Entity) {
	mc.Buffer[mc.Length] = entity
	mc.Length++
}

/*Get - interface implementation */
func (mc *MemoryDBChunk) Get(index int) datastore.Entity {
	return mc.Buffer[index]
}

/*Trim - interface implementation */
func (mc *MemoryDBChunk) Trim() {
	mc.Buffer = mc.Buffer[:mc.Length]
}

/*MemoryDBChunkProvider - a merory db chunk provider */
type MemoryDBChunkProvider struct {
}

/*Create - interface implementation */
func (mcp *MemoryDBChunkProvider) Create(size int) datastore.Chunk {
	c := MemoryDBChunk{}
	c.Buffer = make([]datastore.Entity, size)
	c.Length = 0
	return &c
}

/*MemoryDBChunkProcessor - a chunk processor that stores the entire chunk of entities into the memory db */
type MemoryDBChunkProcessor struct {
	EntityMetadata datastore.EntityMetadata
	ChunkChannel   <-chan datastore.Chunk
}

/*Process - interface implementation */
func (mcp *MemoryDBChunkProcessor) Process(ctx context.Context, chunk datastore.Chunk) {
	mchunk, ok := chunk.(*MemoryDBChunk)
	if !ok {
		panic("invalid chunk into the memory db chunk channel\n")
	}
	lctx := WithEntityConnection(ctx, mcp.EntityMetadata)
	defer Close(lctx)
	store := mcp.EntityMetadata.GetStore()
	err := store.MultiWrite(lctx, mcp.EntityMetadata, mchunk.Buffer)
	if err != nil {
		Logger.Info("memorystore - memory chunk process", zap.Error(err))
	}
}

/*Run - interface implementation */
func (mcp *MemoryDBChunkProcessor) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case chunk := <-mcp.ChunkChannel:
			mcp.Process(ctx, chunk)
		}
	}
}
