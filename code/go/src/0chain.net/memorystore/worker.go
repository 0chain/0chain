package memorystore

import (
	"context"
	"fmt"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*ASYNC_CHANNEL - key used to get the async channel from the context */
const ASYNC_CHANNEL common.ContextKey = "async"

/*WithAsyncChannel takes a context and adds a channel value used for async processing */
func WithAsyncChannel(ctx context.Context, channel chan<- MemoryEntity) context.Context {
	return context.WithValue(ctx, ASYNC_CHANNEL, channel)
}

/*AsyncChannel - Get Async Channel associated with this context */
func AsyncChannel(ctx context.Context) chan<- MemoryEntity {
	async := ctx.Value(ASYNC_CHANNEL)
	if async == nil {
		return nil
	}
	channel, ok := async.(chan<- MemoryEntity)
	if !ok {
		return nil
	}
	return channel
}

/*DoAsyncEntityJSONHandler - a json request response handler that adds a memorystore connection to the Context
* Request is deserialized into an entity
* It reclaims the connection at the end so there is no connection leak
 */
func DoAsyncEntityJSONHandler(handler common.JSONEntityReqResponderF, channel chan<- MemoryEntity) common.JSONEntityReqResponderF {
	return func(ctx context.Context, object interface{}) (interface{}, error) {
		ctx = WithAsyncChannel(ctx, channel)
		entity, err := handler(ctx, object)
		if err != nil {
			return nil, err
		}
		data := make(map[string]interface{})
		data["entity"] = entity
		data["async"] = true
		return data, nil
	}
}

type Chunk struct {
	Buffer []MemoryEntity
	Length int
}

func (c *Chunk) Size() int {
	return c.Length
}

func (c *Chunk) Add(entity MemoryEntity) {
	c.Buffer[c.Length] = entity
	c.Length++
}

func (c *Chunk) Get(index int) MemoryEntity {
	// TODO? Add array checks or assume it's all good for performance?
	return c.Buffer[index]
}

func (c *Chunk) Trim() {
	c.Buffer = c.Buffer[:c.Length]
}

type EntityChunkBuilder struct {
	ChunkSize      int           // Size of the chunks
	MaxHoldupTime  time.Duration // Max holdup time from the first entity added
	EntityChannel  <-chan MemoryEntity
	ChunkChannel   chan<- *Chunk
	TimeoutChannel *time.Timer
	Chunk          *Chunk
}

/*NewChunk - create a new chunk of given size */
func NewChunk(size int) *Chunk {
	c := Chunk{}
	c.Buffer = make([]MemoryEntity, size)
	c.Length = 0
	return &c
}

func (ecb *EntityChunkBuilder) run(ctx context.Context) {
	ecb.Chunk = NewChunk(ecb.ChunkSize)
	for true {
		if ecb.MaxHoldupTime > 0 {
			select {
			case <-ctx.Done():
				ecb.TimeoutChannel.Stop()
				return
			case e := <-ecb.EntityChannel:
				ecb.addEntity(e)
			case _ = <-ecb.TimeoutChannel.C:
				ecb.sendChunk(ecb.ChunkChannel)
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case e := <-ecb.EntityChannel:
				ecb.addEntity(e)
			}
		}
	}
}

func creationDate(entity MemoryEntity) time.Time {
	cf, ok := entity.(datastore.CreationTrackable)
	if ok {
		return time.Unix(int64(cf.GetCreationTime()), 0)
	}
	return time.Now().UTC()
}

func (ecb *EntityChunkBuilder) addEntity(entity MemoryEntity) {
	ecb.Chunk.Add(entity)
	if ecb.Chunk.Size() == ecb.ChunkSize {
		ecb.sendChunk(ecb.ChunkChannel)
		return
	}
	if ecb.MaxHoldupTime > 0 && ecb.Chunk.Size() == 1 {
		delta := creationDate(ecb.Chunk.Get(0)).Add(ecb.MaxHoldupTime).Sub(time.Now().UTC())
		ecb.TimeoutChannel.Reset(delta)
	}
}

func (ecb *EntityChunkBuilder) sendChunk(channel chan<- *Chunk) {
	if ecb.Chunk.Length == 0 {
		return
	}
	ecb.Chunk.Trim()
	channel <- ecb.Chunk
	ecb.Chunk = NewChunk(ecb.ChunkSize)
}

type ChunkStorer struct {
	ChunkChannel <-chan *Chunk
}

func (bs *ChunkStorer) run(ctx context.Context) {
	// TODO: What happens if a connection expires? We need a way to catch exception and get a new connection
	lctx := WithConnection(ctx)
	defer GetCon(lctx).Close()
	for true {
		select {
		case <-ctx.Done():
			return
		case chunk := <-bs.ChunkChannel:
			err := MultiWrite(lctx, chunk.Buffer)
			if err != nil {
				fmt.Printf("multiwrite error : %v\n", err)
			}
		}
	}
}

/*SetupWorkers - This setups up workers that allows aggregating and storing entities in chunks */
func SetupWorkers(ctx context.Context, options *CollectionOptions) chan MemoryEntity {
	echannel := make(chan MemoryEntity, options.EntityBufferSize)
	bchannel := make(chan *Chunk, options.ChunkBufferSize)
	var ecb EntityChunkBuilder
	ecb.ChunkSize = options.ChunkSize
	ecb.MaxHoldupTime = options.MaxHoldupTime
	ecb.EntityChannel = echannel
	ecb.ChunkChannel = bchannel
	ecb.TimeoutChannel = time.NewTimer(-100 * time.Second)
	bworkers := make([]ChunkStorer, options.NumChunkStorers)
	for _, bworker := range bworkers {
		bworker.ChunkChannel = bchannel
		go bworker.run(ctx)
	}
	go ecb.run(ctx)
	return echannel
}
