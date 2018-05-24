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

func NewChunk(size int) *Chunk {
	c := Chunk{}
	c.Buffer = make([]Entity, size)
	c.Length = 0
	return &c
}

func (ebb *EntityChunkBuilder) run() {
	ebb.Chunk = NewChunk(ebb.ChunkSize)
	for true {
		if ebb.MaxHoldupTime > 0 {
			select {
			case e := <-ebb.EntityChannel:
				ebb.addEntity(e)
			case _ = <-ebb.TimeoutChannel.C:
				ebb.sendChunk(ebb.ChunkChannel)
			}
		} else {
			e := <-ebb.EntityChannel
			ebb.addEntity(e)
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

func (ebb *EntityChunkBuilder) addEntity(entity Entity) {
	ebb.Chunk.Add(entity)
	if ebb.Chunk.Size() == ebb.ChunkSize {
		ebb.sendChunk(ebb.ChunkChannel)
		return
	}
	if ebb.MaxHoldupTime > 0 && ebb.Chunk.Size() == 1 {
		delta := creationDate(ebb.Chunk.Get(0)).Add(ebb.MaxHoldupTime).Sub(time.Now())
		ebb.TimeoutChannel.Reset(delta)
	}
}

func (ebb *EntityChunkBuilder) sendChunk(channel chan<- *Chunk) {
	if ebb.Chunk.Length == 0 {
		return
	}
	ebb.Chunk.Trim()
	channel <- ebb.Chunk
	ebb.Chunk = NewChunk(ebb.ChunkSize)
}

type ChunkStorer struct {
	ChunkChannel <-chan *Chunk
}

func (bs *ChunkStorer) run() {
	// TODO: What happens if a connection expires? We need a way to catch exception and get a new connection
	ctx := WithConnection(context.Background())
	defer GetCon(ctx).Close()
	for true {
		chunk := <-bs.ChunkChannel
		err := MultiWrite(ctx, chunk.Buffer)
		if err != nil {
			fmt.Printf("multiwrite error : %v\n", err)
		}
	}
}

/*SetupWorkers - This setups up workers that allows aggregating and storing entities in chunks */
func SetupWorkers(entityBufferSize int, maxHoldupTime time.Duration, chunkSize int, chunkBufferSize int, numChunkWorkers int) chan Entity {
	echannel := make(chan Entity, entityBufferSize)
	bchannel := make(chan *Chunk, chunkBufferSize)
	var ebb EntityChunkBuilder
	ebb.ChunkSize = chunkSize
	ebb.MaxHoldupTime = maxHoldupTime
	ebb.EntityChannel = echannel
	ebb.ChunkChannel = bchannel
	ebb.TimeoutChannel = time.NewTimer(-100 * time.Second)

	bworkers := make([]ChunkStorer, numChunkWorkers)
	for _, bworker := range bworkers {
		bworker.ChunkChannel = bchannel
		go bworker.run()
	}
	go ebb.run()
	return echannel
}
