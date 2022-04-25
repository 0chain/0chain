package datastore

import (
	"context"
	"time"

	"0chain.net/core/common"
)

/*SetupChunkWorkers - Start the chunk build and processor workers */
func SetupChunkWorkers(ctx context.Context, ecb EntityChunkBuilder, ecpWorkers []ChunkProcessor) {
	for _, bworker := range ecpWorkers {
		go bworker.Run(ctx)
	}
	go ecb.Run(ctx)
}

type QueuedEntity struct {
	Entity     Entity
	QueuedTime time.Time
}

/*ASYNC_CHANNEL - key used to get the async channel from the context */
const ASYNC_CHANNEL common.ContextKey = "async"

/*WithAsyncChannel takes a context and adds a channel value used for async processing */
func WithAsyncChannel(ctx context.Context, channel chan<- QueuedEntity) context.Context {
	return context.WithValue(ctx, ASYNC_CHANNEL, channel)
}

/*AsyncChannel - Get Async Channel associated with this context */
func AsyncChannel(ctx context.Context) chan<- QueuedEntity {
	async := ctx.Value(ASYNC_CHANNEL)
	if async == nil {
		return nil
	}
	channel, ok := async.(chan<- QueuedEntity)
	if !ok {
		return nil
	}
	return channel
}

func DoAsync(ctx context.Context, entity Entity) bool {
	channel := AsyncChannel(ctx)
	if channel != nil {
		qe := QueuedEntity{Entity: entity, QueuedTime: time.Now()}
		channel <- qe
		return true
	}
	return false
}

/*DoAsyncEntityJSONHandler - a json request response handler that adds a memorystore connection to the Context
* Request is deserialized into an entity
* It reclaims the connection at the end so there is no connection leak
 */
func DoAsyncEntityJSONHandler(handler JSONEntityReqResponderF, channel chan<- QueuedEntity) JSONEntityReqResponderF {
	return func(ctx context.Context, entity Entity) (interface{}, error) {
		ctx = WithAsyncChannel(ctx, channel)
		rentity, err := handler(ctx, entity)
		if err != nil {
			return nil, err
		}
		data := make(map[string]interface{})
		data["entity"] = rentity
		data["async"] = true
		return data, nil
	}
}

type Chunk interface {
	Size() int
	Add(entity Entity)
	Get(index int) Entity
	Trim()
}

/*ChunkingOptions - to tune the performance charactersistics of async batch writing */
type ChunkingOptions struct {
	EntityMetadata   EntityMetadata
	EntityBufferSize int
	MaxHoldupTime    time.Duration
	NumChunkCreators int
	ChunkSize        int
	ChunkBufferSize  int
	NumChunkStorers  int
}

type ChunkProvider interface {
	Create(size int) Chunk
}

type EntityChunkBuilder struct {
	ChunkSize      int           // Size of the chunks
	MaxHoldupTime  time.Duration // Max holdup time from the first entity added
	EntityChannel  <-chan QueuedEntity
	ChunkChannel   chan<- Chunk
	TimeoutChannel *time.Timer
	Chunk          Chunk
	ChunkProvider  ChunkProvider
}

func (ecb *EntityChunkBuilder) Run(ctx context.Context) {
	ecb.Chunk = ecb.ChunkProvider.Create(ecb.ChunkSize)
	for {
		if ecb.MaxHoldupTime > 0 {
			select {
			case <-ctx.Done():
				ecb.TimeoutChannel.Stop()
				return
			case qe := <-ecb.EntityChannel:
				ecb.addEntity(qe)
			case <-ecb.TimeoutChannel.C:
				ecb.sendChunk(ecb.ChunkChannel)
			}
		} else {
			select {
			case <-ctx.Done():
				return
			case qe := <-ecb.EntityChannel:
				ecb.addEntity(qe)
			}
		}
	}
}

func (ecb *EntityChunkBuilder) addEntity(qe QueuedEntity) {
	ecb.Chunk.Add(qe.Entity)
	if ecb.Chunk.Size() == ecb.ChunkSize {
		ecb.sendChunk(ecb.ChunkChannel)
		return
	}
	if ecb.MaxHoldupTime > 0 && ecb.Chunk.Size() == 1 {
		end := qe.QueuedTime.Add(ecb.MaxHoldupTime)
		delta := time.Until(end)
		if delta > 0 {
			ecb.TimeoutChannel.Reset(delta)
		}
	}
}

func (ecb *EntityChunkBuilder) sendChunk(channel chan<- Chunk) {
	if ecb.Chunk.Size() == 0 {
		return
	}
	ecb.Chunk.Trim()
	channel <- ecb.Chunk
	ecb.Chunk = ecb.ChunkProvider.Create(ecb.ChunkSize)
}

type ChunkProcessor interface {
	Run(ctx context.Context)
	Process(ctx context.Context, chunk Chunk)
}
