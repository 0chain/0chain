package memorystore

import (
	"context"
	"fmt"
	"time"

	"0chain.net/datastore"
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

type MemoryDBChunk struct {
	Buffer []datastore.Entity
	Length int
}

func (mc *MemoryDBChunk) Size() int {
	return mc.Length
}

func (mc *MemoryDBChunk) Add(entity datastore.Entity) {
	mc.Buffer[mc.Length] = entity
	mc.Length++
}

func (mc *MemoryDBChunk) Get(index int) datastore.Entity {
	// TODO? Add array checks or assume it's all good for performance?
	return mc.Buffer[index]
}

func (mc *MemoryDBChunk) Trim() {
	mc.Buffer = mc.Buffer[:mc.Length]
}

/*MemoryDBChunkProvider - a merory db chunk provider */
type MemoryDBChunkProvider struct {
}

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

func (mcp *MemoryDBChunkProcessor) Process(ctx context.Context, chunk datastore.Chunk) {
	mchunk, ok := chunk.(*MemoryDBChunk)
	if !ok {
		panic("invalid chunk into the memory db chunk channel\n")
	}
	store := mcp.EntityMetadata.GetStore()
	err := store.MultiWrite(ctx, mcp.EntityMetadata, mchunk.Buffer)
	if err != nil {
		fmt.Printf("multiwrite error : %v\n", err)
	}
}

func (mcp *MemoryDBChunkProcessor) Run(ctx context.Context) {
	// TODO: What happens if a connection expires? We need a way to catch exception and get a new connection
	lctx := WithEntityConnection(ctx, mcp.EntityMetadata)
	defer Close(lctx)
	for true {
		select {
		case <-ctx.Done():
			return
		case chunk := <-mcp.ChunkChannel:
			mcp.Process(lctx, chunk)
		}
	}
}
