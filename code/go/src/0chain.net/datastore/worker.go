package datastore

import (
	"context"
	"fmt"
	"time"

	"0chain.net/common"
)

/*ASYNC_CHANNEL - key used to get the async channel from the context */
const ASYNC_CHANNEL common.ContextKey = "async"

/*WithAsyncChannel takes a context and adds a channel value used for async processing */
func WithAsyncChannel(ctx context.Context, channel chan<- Entity) context.Context {
	return context.WithValue(ctx, ASYNC_CHANNEL, channel)
}

/*AsyncChannel - Get Async Channel associated with this context */
func AsyncChannel(ctx context.Context) chan<- Entity {
	async := ctx.Value(ASYNC_CHANNEL)
	if async == nil {
		return nil
	}
	channel, ok := async.(chan<- Entity)
	if !ok {
		return nil
	}
	return channel
}

/*DoAsyncEntityJSONHandler - a json request response handler that adds a datastore connection to the Context
* Request is deserialized into an entity
* It reclaims the connection at the end so there is no connection leak
 */
func DoAsyncEntityJSONHandler(handler common.JSONEntityReqResponderF, channel chan<- Entity) common.JSONEntityReqResponderF {
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
	Buffer []Entity
	Length int
}

func (c *Chunk) Size() int {
	return c.Length
}

func (c *Chunk) Add(entity Entity) {
	c.Buffer[c.Length] = entity
	c.Length++
}

func (c *Chunk) Get(index int) Entity {
	// TODO? Add array checks or assume it's all good for performance?
	return c.Buffer[index]
}

func (c *Chunk) Trim() {
	c.Buffer = c.Buffer[:c.Length]
}

type EntityChunkBuilder struct {
	ChunkSize      int           // Size of the chunks
	MaxHoldupTime  time.Duration // Max holdup time from the first entity added
	EntityChannel  <-chan Entity
	ChunkChannel   chan<- *Chunk
	TimeoutChannel *time.Timer
	Chunk          *Chunk
}

/*NewChunk - create a new chunk of given size */
func NewChunk(size int) *Chunk {
	c := Chunk{}
	c.Buffer = make([]Entity, size)
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

func creationDate(entity Entity) common.Time {
	cf, ok := entity.(CreationTrackable)
	if ok {
		return cf.GetCreationTime()
	}
	return common.Now()
}

func (ecb *EntityChunkBuilder) addEntity(entity Entity) {
	ecb.Chunk.Add(entity)
	if ecb.Chunk.Size() == ecb.ChunkSize {
		ecb.sendChunk(ecb.ChunkChannel)
		return
	}
	if ecb.MaxHoldupTime > 0 && ecb.Chunk.Size() == 1 {
		delta := creationDate(ecb.Chunk.Get(0)).Add(ecb.MaxHoldupTime).Sub(time.Now())
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
func SetupWorkers(ctx context.Context, entityBufferSize int, maxHoldupTime time.Duration, chunkSize int, chunkBufferSize int, numChunkWorkers int) chan Entity {
	echannel := make(chan Entity, entityBufferSize)
	bchannel := make(chan *Chunk, chunkBufferSize)
	var ecb EntityChunkBuilder
	ecb.ChunkSize = chunkSize
	ecb.MaxHoldupTime = maxHoldupTime
	ecb.EntityChannel = echannel
	ecb.ChunkChannel = bchannel
	ecb.TimeoutChannel = time.NewTimer(-100 * time.Second)
	bworkers := make([]ChunkStorer, numChunkWorkers)
	for _, bworker := range bworkers {
		bworker.ChunkChannel = bchannel
		go bworker.run(ctx)
	}
	go ecb.run(ctx)
	return echannel
}
