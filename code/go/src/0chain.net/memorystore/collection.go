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

/*BATCH_SIZE size of the batch */
const BATCH_SIZE = 256

/*IterateCollection - iterate a collection with a callback that is given the entities.
*Iteration can be stopped by returning false
 */
func (ms *Store) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	con := GetEntityCon(ctx, entityMetadata)
	bucket := make([]datastore.Entity, BATCH_SIZE)
	keys := make([]datastore.Key, BATCH_SIZE)
	maxscore := math.MaxInt64
	offset := 0
	proceed := true
	for idx := 0; true; idx += BATCH_SIZE {
		select {
		case <-ctx.Done():
			return common.ErrStop
		default:
		}
		con.Send("ZREVRANGEBYSCORE", collectionName, maxscore, 0, "LIMIT", offset, BATCH_SIZE)
		con.Flush()
		data, err := con.Receive()
		if err != nil {
			return err
		}
		bkeys, ok := data.([]interface{})
		if len(bkeys) == 0 {
			return nil
		}
		// wonder if WITHSCORES and adjusting the maxscore is more performant rather than adjusting offest
		offset += len(bkeys)
		if !ok {
			return common.NewError("error", fmt.Sprintf("error casting data to []interface{} : %T", data))
		}
		for bidx := range bkeys {
			bucket[bidx] = entityMetadata.Instance().(MemoryEntity)
			keys[bidx] = datastore.ToKey(bkeys[bidx])
		}

		err = ms.MultiRead(ctx, entityMetadata, keys[:len(bkeys)], bucket)
		if err != nil {
			return err
		}
		for idx := range bkeys {
			proceed = handler(ctx, bucket[idx].(datastore.CollectionEntity))
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
func PrintIterator(ctx context.Context, qe datastore.CollectionEntity) bool {
	fmt.Printf("pi: %v\n", qe)
	return true
}

var collections = make(map[string]bool)
var collectionsMutex = &sync.Mutex{}

func trackCollection(entityMetadata datastore.EntityMetadata, qe datastore.CollectionEntity) {
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
	go CollectionTrimmer(entityMetadata, qe.GetCollectionName(), qe.GetCollectionSize(), qe.GetCollectionDuration())
	collections[qe.GetCollectionName()] = true
}

/*CollectionTrimmer - trims the collection based on size and duration options */
func CollectionTrimmer(entityMetadata datastore.EntityMetadata, collection string, trimSize int64, trimBeyond time.Duration) {
	fmt.Printf("starting collection trimmer for %v\n", collection)
	ctx := WithEntityConnection(common.GetRootContext(), entityMetadata)
	con := GetEntityCon(ctx, entityMetadata)
	defer Close(ctx)
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
			score := datastore.GetCollectionScore(time.Now().Add(-trimBeyond))
			con.Send("ZREMRANGEBYSCORE", collection, 0, score)
		}
	}
}
