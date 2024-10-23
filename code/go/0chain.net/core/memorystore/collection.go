package memorystore

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"sync"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
	. "github.com/0chain/common/core/logging"
	"go.uber.org/zap"
)

/*IterateCollection - iterate a collection with a callback that is given the entities.
*Iteration can be stopped by returning false
 */
func (ms *Store) IterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	return ms.iterateCollection(ctx, entityMetadata, collectionName, datastore.Descending, handler)
}

/*IterateCollectionAsc - iterate a collection in ascedning order with a callback that is given the entities.
*Iteration can be stopped by returning false
 */
func (ms *Store) IterateCollectionAsc(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, handler datastore.CollectionIteratorHandler) error {
	return ms.iterateCollection(ctx, entityMetadata, collectionName, datastore.Ascending, handler)
}

func (ms *Store) iterateCollection(ctx context.Context, entityMetadata datastore.EntityMetadata, collectionName string, order datastore.Order, handler datastore.CollectionIteratorHandler) error {
	con := GetEntityCon(ctx, entityMetadata)
	bucket := make([]datastore.Entity, BATCH_SIZE)
	keys := make([]datastore.Key, BATCH_SIZE)
	var maxscore int64 = math.MaxInt64
	var minscore int64 = math.MinInt64
	offset := 0
	proceed := true
	selectCommand := "ZREVRANGEBYSCORE"
	if order == datastore.Ascending {
		selectCommand = "ZRANGEBYSCORE"
		maxscore, minscore = minscore, maxscore
	}
	ckeys := make(map[datastore.Key]bool)
	for idx := 0; true; idx += BATCH_SIZE {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		err := con.Send(selectCommand, collectionName, maxscore, minscore, "WITHSCORES", "LIMIT", offset, BATCH_SIZE)
		if err != nil {
			return err
		}
		if err := con.Flush(); err != nil {
			return err
		}
		data, err := con.Receive()
		if err != nil {
			return err
		}
		bkeys, ok := data.([]interface{})
		count := len(bkeys) / 2
		if count == 0 {
			Logger.Info("Redis returned 0 rows after seclect")
			return nil
		}
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
				score, err := strconv.ParseInt(string(scoredata), 10, 64)
				if err != nil {
					Logger.Debug("iterator error", zap.ByteString("score", scoredata), zap.String("type", fmt.Sprintf("%T", bkeys[2*i+1])))
					return err
				}
				ce.SetCollectionScore(score)
			} else {
				Logger.Info("iterator error", zap.Any("score", bkeys[2*i+1]), zap.String("type", fmt.Sprintf("%T", bkeys[2*i+1])))
			}
		}
		err = ms.MultiRead(ctx, entityMetadata, keys[:count], bucket)
		if err != nil {
			return err
		}
		for i := 0; i < count; i++ {
			/*
			* Adding key to entity instance that has hash
			* in collection, but no corresponding entity.
			* This allows handler to process entities that
			* only appear in the collection.
			 */
			if bucket[i].GetKey() == "" {
				bucket[i].SetKey(keys[i])
			}
			if datastore.IsEmpty(bucket[i].GetKey()) {
				continue
			}
			if e, ok := ckeys[bucket[i].GetKey()]; ok {
				continue
			} else {
				ckeys[bucket[i].GetKey()] = e
			}
			proceed, err = handler(ctx, bucket[i].(datastore.CollectionEntity))
			if err != nil {
				return err
			}

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
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if err := con.Send("ZCARD", collection); err != nil {
				Logger.Error("collection trimmer", zap.String("collection", collection), zap.Time("time", t), zap.Error(err))
				continue
			}
			if err := con.Flush(); err != nil {
				Logger.Error("collection trimmer", zap.String("collection", collection), zap.Time("time", t), zap.Error(err))
				continue
			}
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
			if err := con.Send("ZREMRANGEBYSCORE", collection, 0, score); err != nil {
				Logger.Error("collection trimmer", zap.String("collection", collection), zap.Time("time", t), zap.Error(err))
				continue
			}
		}
	}
}
