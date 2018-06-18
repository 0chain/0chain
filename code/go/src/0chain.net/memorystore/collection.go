package memorystore

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"0chain.net/common"
	"0chain.net/datastore"
	. "0chain.net/logging"
	"go.uber.org/zap"
)

/*IterateCollection - iterate a collection with a callback that is given the entities.
*Iteration can be stopped by returning false
 */
func (ms *Store) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	con := GetEntityCon(ctx, entityMetadata)
	bucket := make([]datastore.Entity, BATCH_SIZE)
	keys := make([]datastore.Key, BATCH_SIZE)
	maxscore := math.MaxInt64
	minscore := math.MinInt64
	offset := 0
	proceed := true
	for idx := 0; true; idx += BATCH_SIZE {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		con.Send("ZREVRANGEBYSCORE", collectionName, maxscore, minscore, "WITHSCORES", "LIMIT", offset, BATCH_SIZE)
		con.Flush()
		data, err := con.Receive()
		if err != nil {
			return err
		}
		bkeys, ok := data.([]interface{})
		count := len(bkeys) / 2
		if count == 0 {
			return nil
		}
		// TODO: wonder if WITHSCORES and adjusting the maxscore is more performant rather than adjusting offest
		offset += count
		if !ok {
			return common.NewError("error", fmt.Sprintf("error casting data to []interface{} : %T", data))
		}
		for i := 0; i < count; i++ {
			bucket[i] = entityMetadata.Instance()
			keys[i] = datastore.ToKey(bkeys[2*i])
			ce := bucket[i].(datastore.CollectionEntity)
			scoredata, ok := bkeys[2*i+1].([]byte)
			if ok {
				score, err := strconv.ParseInt(string(scoredata), 10, 63)
				if err != nil {
					Logger.Debug("iterator error", zap.Any("score", scoredata), zap.Any("type", fmt.Sprintf("%T", bkeys[2*i+1])))
					return err
				}
				ce.SetCollectionScore(score)
			} else {
				Logger.Debug("iterator error", zap.Any("score", bkeys[2*i+1]), zap.Any("type", fmt.Sprintf("%T", bkeys[2*i+1])))
			}
		}

		err = ms.MultiRead(ctx, entityMetadata, keys[:count], bucket)
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			proceed = handler(ctx, bucket[i].(datastore.CollectionEntity))
			if !proceed {
				break
			}
		}
		if !proceed {
			break
		}
		if count < BATCH_SIZE {
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
	collectionsMutex.Lock()
	defer collectionsMutex.Unlock()
	_, ok := collections[qe.GetCollectionName()]
	if ok {
		return
	}
	go CollectionTrimmer(entityMetadata, qe.GetCollectionName(), qe.GetCollectionSize(), qe.GetCollectionDuration())
	collections[qe.GetCollectionName()] = true
}

/*CollectionTrimmer - trims the collection based on size and duration options */
func CollectionTrimmer(entityMetadata datastore.EntityMetadata, collection string, trimSize int64, trimBeyond time.Duration) {
	Logger.Debug("starting collection trimmer", zap.String("collection", collection))
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
				Logger.Error("collection trimmer", zap.String("collection", collection), zap.Time("time", t), zap.Error(err))
				continue
			}
			size, ok := data.(int64)
			if !ok {
				Logger.Error("collection trimmer", zap.String("collection", collection), zap.Time("time", t), zap.Any("data", data))
			}
			if size < trimSize {
				continue
			}
			score := datastore.GetCollectionScore(time.Now().Add(-trimBeyond))
			con.Send("ZREMRANGEBYSCORE", collection, 0, score)
		}
	}
}
