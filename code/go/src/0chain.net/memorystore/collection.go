package memorystore

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
)

/*CollectionOptions - to tune the performance charactersistics of a collection */
type CollectionOptions struct {
	EntityBufferSize int
	MaxHoldupTime    time.Duration
	NumChunkCreators int
	ChunkSize        int
	ChunkBufferSize  int
	NumChunkStorers  int
}

/*CollectionEntity - An entity can implement the CollectionEntity interface by including a CollectionIDField
*It can optionally override GetCollectionName to provide multiple collections partitioned by some key
* Example - transactions and blocks can be partioned by chain
 */
type CollectionEntity interface {
	MemoryEntity
	GetCollectionName() string
	GetCollectionSize() int64
	GetCollectionDuration() time.Duration
	InitCollectionScore()
	SetCollectionScore(score int64)
	GetCollectionScore() int64 // larger scores have higher priority
	AddToCollection(ctx context.Context, collectionName string) error
}

/*EntityCollection - Entities can be organized into collections. EntityCollection provides configuration for those collections */
type EntityCollection struct {
	CollectionName     string
	CollectionSize     int64
	CollectionDuration time.Duration
}

/*GetCollectionName - Given an partitioning key (such as parent key), returns the key for the collection */
func (eq *EntityCollection) GetCollectionName(parent datastore.Key) string {
	if datastore.IsEmpty(parent) {
		return eq.CollectionName
	}
	return fmt.Sprintf("%s:%s", eq.CollectionName, parent)
}

/*CollectionIDField - An entity with a CollectionIDField will automatically put that entity into a collection */
type CollectionIDField struct {
	datastore.IDField
	EntityCollection *EntityCollection `json:"-"`
	CollectionScore  int64             `json:"-"`
}

/*GetCollectionName - default implementation for CollectionEntity interface
* Entities can override this method to provide collections partitioned by some key
**/
func (cf *CollectionIDField) GetCollectionName() string {
	return cf.EntityCollection.CollectionName
}

func (cf *CollectionIDField) GetCollectionSize() int64 {
	return cf.EntityCollection.CollectionSize
}

func (cf *CollectionIDField) GetCollectionDuration() time.Duration {
	return cf.EntityCollection.CollectionDuration
}

/*GetCollectionScore - override */
func (cf *CollectionIDField) GetCollectionScore() int64 {
	return cf.CollectionScore
}

/*SetCollectionScore - override */
func (cf *CollectionIDField) SetCollectionScore(score int64) {
	cf.CollectionScore = score
}

/*InitCollectionScore - override */
func (cf *CollectionIDField) InitCollectionScore() {
	cf.SetCollectionScore(getScore(time.Now()))
}
func getScore(ts time.Time) int64 {
	// score := time.Now().UniqNano() // nano seconds (10^18)
	// score := time.Now().Unix() // seconds (10^9)
	return ts.UnixNano() / int64(time.Millisecond) // 10^12
}

/*AddToCollection - default implementation for CollectionEntity interface */
func (cf *CollectionIDField) AddToCollection(ctx context.Context, collectionName string) error {
	con := GetCon(ctx)
	con.Send("ZADD", collectionName, cf.GetCollectionScore(), cf.GetKey())
	con.Flush()
	_, err := con.Receive()
	if err != nil {
		return err
	}
	return nil
}

/*MultiAddToCollection adds multiple entities to a collection */
func MultiAddToCollection(ctx context.Context, entities []MemoryEntity) error {
	// Assuming all entities belong to the same collection.
	if len(entities) == 0 {
		return nil
	}
	svpair := make([]interface{}, 1+2*len(entities))
	ce := entities[0].(CollectionEntity)
	trackCollection(ce)
	svpair[0] = ce.GetCollectionName()
	offset := 1
	for idx, entity := range entities {
		ce, ok := entity.(CollectionEntity)
		if !ok {
			return common.NewError("dev_error", "Entity needs to be CollectionEntity")
		}
		ind := offset + 2*idx
		score := ce.GetCollectionScore()
		if score == 0 {
			ce.InitCollectionScore()
		}
		svpair[ind] = ce.GetCollectionScore()
		svpair[ind+1] = ce.GetKey()
	}
	con := GetCon(ctx)
	con.Send("ZADD", svpair...)
	con.Flush()
	_, err := con.Receive()
	return err
}

/*CollectionIteratorHandler is a collection iteration handler function type */
type CollectionIteratorHandler func(ctx context.Context, ce CollectionEntity) bool

/*BATCH_SIZE size of the batch */
const BATCH_SIZE = 100

/*IterateCollection - iterate a collection with a callback that is given the entities.
*Iteration can be stopped by returning false
 */
func IterateCollection(ctx context.Context, collectionName string, handler CollectionIteratorHandler, entityProvider common.EntityProvider) error {
	con := GetCon(ctx)
	bucket := make([]MemoryEntity, BATCH_SIZE)
	keys := make([]datastore.Key, BATCH_SIZE)
	maxscore := math.MaxInt64
	offset := 0
	proceed := true
	for idx := 0; true; idx += BATCH_SIZE {
		con.Send("ZREVRANGEBYSCORE", collectionName, maxscore, 0, "LIMIT", offset, BATCH_SIZE)
		con.Flush()
		data, err := con.Receive()
		if err != nil {
			return err
		}
		bkeys, ok := data.([]interface{})
		// wonder if WITHSCORES and adjusting the maxscore is more performant rather than adjusting offest
		offset += len(bkeys)
		if !ok {
			return common.NewError("error", fmt.Sprintf("error casting data to []interface{} : %T", data))
		}
		for bidx := range bkeys {
			bucket[bidx] = entityProvider().(MemoryEntity)
			keys[bidx] = datastore.ToKey(bkeys[bidx])
		}

		err = MultiRead(ctx, keys[:len(bkeys)], bucket)
		if err != nil {
			return err
		}
		for idx := range bkeys {
			proceed = handler(ctx, bucket[idx].(CollectionEntity))
			if !proceed {
				break
			}
		}
		if !proceed {
			break
		}
		if len(bkeys) < BATCH_SIZE {
			break
		}
	}
	return nil
}

/*PrintIterator - a simple iterator handler that just prints the entity */
func PrintIterator(ctx context.Context, qe CollectionEntity) bool {
	fmt.Printf("pi: %v\n", qe)
	return true
}

var collections = make(map[string]bool)
var collectionsMutex = &sync.Mutex{}

func trackCollection(qe CollectionEntity) {
	_, ok := collections[qe.GetCollectionName()]
	if ok {
		return
	}
	collectionsMutex.Lock()
	defer collectionsMutex.Unlock()
	_, ok = collections[qe.GetCollectionName()]
	if ok {
		return
	}
	go CollectionTrimmer(qe.GetCollectionName(), qe.GetCollectionSize(), qe.GetCollectionDuration())
	collections[qe.GetCollectionName()] = true
}

func CollectionTrimmer(collection string, trimSize int64, trimBeyond time.Duration) {
	fmt.Printf("starting collection trimmer for %v\n", collection)
	ctx := WithConnection(common.GetRootContext())
	con := GetCon(ctx)
	defer con.Close()
	ticker := time.NewTicker(trimBeyond)
	for true {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			con.Send("ZCARD", collection)
			con.Flush()
			data, err := con.Receive()
			if err != nil {
				fmt.Printf("collection trimmer %v %v error: %v\n", t, collection, err)
				continue
			}
			size, ok := data.(int64)
			if !ok {
				fmt.Printf("collection trimmer %v %v data: %v\n", t, collection, data)
			}
			if size < trimSize {
				continue
			}
			score := getScore(time.Now().Add(-trimBeyond))
			con.Send("ZREMRANGEBYSCORE", collection, 0, score)
		}
	}
}
