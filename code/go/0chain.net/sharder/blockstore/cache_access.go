package blockstore

import (
	"context"
	"fmt"
	"strings"
	"time"

	"0chain.net/core/common"
	"0chain.net/core/datastore"
)

// Add a cache bucket to store accessed time as key and hash as its value
// eg accessedTime:hash
// Use sorting feature of boltdb to quickly Delete cached blocks that should be replaced
type cacheAccess struct {
	Hash          string     `json:"hash"`
	AccessTime    *time.Time `json:"accessTime"`
	accessTimeKey string
	datastore.CollectionMemberField
}

func NewCacheAccess(hash string, accessTime *time.Time) *cacheAccess {
	ca := &cacheAccess{Hash: hash, AccessTime: accessTime}
	timeStr := accessTime.Format(time.RFC3339)
	accessTimeKey := fmt.Sprintf("%v%v%v", timeStr, cacheAccessTimeSeparator, ca.Hash)
	ca.SetKey(accessTimeKey)
	ca.EntityCollection = cacheAccessEntityCollection
	return ca
}

func DefaultCacheAccess() *cacheAccess {
	ca := &cacheAccess{}
	ca.EntityCollection = cacheAccessEntityCollection
	return ca
}

func (ca *cacheAccess) GetEntityMetadata() datastore.EntityMetadata {
	return cacheAccessEntityMetadata
}

func (ca *cacheAccess) SetKey(key datastore.Key) {
	ca.accessTimeKey = key
}

func (ca *cacheAccess) GetKey() datastore.Key {
	return ca.accessTimeKey
}

func (ca *cacheAccess) GetScore() int64 {
	return ca.GetCollectionScore()
}

func (ca *cacheAccess) ComputeProperties() {
	// Not implemented
}

func (ca *cacheAccess) Validate(ctx context.Context) error {
	return nil // Not implemented
}

func (ca *cacheAccess) Read(ctx context.Context, key datastore.Key) error {
	return nil // Not implemented
}

func (ca *cacheAccess) Write(ctx context.Context) error {
	return nil // Not implemented
}

func (ca *cacheAccess) Delete(context context.Context) error {
	ctx := ca.GetEntityMetadata().GetStore().StartTx(context, ca)

	err := ca.GetEntityMetadata().GetStore().DeleteFromCollection(context, ca)
	if err != nil {
		return err
	}

	err = ca.GetEntityMetadata().GetStore().HDel(ctx, ca, hashCacheHashAccessTime, ca.Hash)
	if err != nil {
		return err
	}

	err = ca.GetEntityMetadata().GetStore().SendTX(ctx, ca)
	if err != nil {
		return err
	}

	return nil
}

func GetHashKeysForReplacement() chan *cacheAccess {
	ch := make(chan *cacheAccess, 10)
	cache := DefaultCacheAccess()
	go func() {
		defer func() {
			close(ch)
		}()

		store := cache.GetEntityMetadata().GetStore()
		collectionName := cache.GetCollectionName()
		count := store.GetCollectionSize(common.GetRootContext(), cache.GetEntityMetadata(), collectionName)
		count /= 2 // Number of blocks to replace
		var endRange int64 = 1000
		k := count
		for i := 0; i < int(count); i += int(endRange) {
			if endRange > k {
				endRange = k
			} else {
				k -= endRange
			}

			var entities []datastore.Entity
			err := store.GetRangeFromCollection(
				common.GetRootContext(),
				cache,
				entities,
				false,
				false,
				string(0),
				string(endRange),
				0,
				0,
			)
			if err != nil {
				return
			}
			for _, e := range entities {
				block := e.GetKey()
				ca := new(cacheAccess)
				sl := strings.Split(block, cacheAccessTimeSeparator)
				ca.Hash = sl[1]
				ch <- ca
			}
		}
	}()

	return ch
}

func (ca *cacheAccess) addOrUpdate() error {
	timeStr := ca.AccessTime.Format(time.RFC3339)
	accessTimeKey := fmt.Sprintf("%v%v%v", timeStr, cacheAccessTimeSeparator, ca.Hash)
	rctx := common.GetRootContext()

	timeValue, err := ca.GetEntityMetadata().GetStore().HGet(rctx, ca, hashCacheHashAccessTime, ca.Hash)
	if err != nil {
		return err
	}
	if timeValue != "" {
		delKey := fmt.Sprintf("%v%v%v", timeValue, cacheAccessTimeSeparator, ca.Hash)
		ca.SetKey(delKey)
		err = ca.GetEntityMetadata().GetStore().DeleteFromCollection(rctx, ca)
		if err != nil {
			return err
		}
	}

	ctx := ca.GetEntityMetadata().GetStore().StartTx(rctx, ca)
	ca.SetKey(accessTimeKey)
	ca.SetCollectionScore(0)
	err = ca.GetEntityMetadata().GetStore().AddToCollection(ctx, ca)
	if err != nil {
		return err
	}
	err = ca.GetEntityMetadata().GetStore().HSet(ctx, ca, hashCacheHashAccessTime, ca.Hash, timeStr)
	if err != nil {
		return err
	}
	err = ca.GetEntityMetadata().GetStore().SendTX(ctx, ca)
	if err != nil {
		return err
	}

	return nil
}

// func (ca *cacheAccess) update() {
// 	timeStr := ca.AccessTime.Format(time.RFC3339)
// 	accessTimeKey := []byte(fmt.Sprintf("%v%v%v", timeStr, cacheAccessTimeSeparator, ca.Hash))

// 	bwrDB.Update(func(t *bbolt.Tx) error {
// 		accessTimeBkt := t.Bucket([]byte(CacheAccessTimeHashBucket))
// 		if accessTimeBkt == nil {
// 			return fmt.Errorf("%v bucket does not exist", CacheAccessTimeHashBucket)
// 		}

// 		hashBkt := t.Bucket([]byte(CacheHashAccessTimeBucket))
// 		if hashBkt == nil {
// 			return fmt.Errorf("%v bucket does not exist", CacheHashAccessTimeBucket)
// 		}

// 		oldAccessTime := hashBkt.Get([]byte(ca.Hash))
// 		if oldAccessTime != nil {
// 			k := []byte(fmt.Sprintf("%v%v%v", string(oldAccessTime), cacheAccessTimeSeparator, ca.Hash))
// 			accessTimeBkt.Delete(k)
// 		}

// 		if err := hashBkt.Put([]byte(ca.Hash), []byte(timeStr)); err != nil {
// 			return err
// 		}
// 		return accessTimeBkt.Put(accessTimeKey, nil)
// 	})
// }

var cacheAccessEntityMetadata *datastore.EntityMetadataImpl

// ProviderCacheAccess - entity provider for client object
func ProviderCacheAccess() datastore.Entity {
	b := &cacheAccess{}
	b.EntityCollection = cacheAccessEntityCollection
	return b
}

// setupEntityCacheAccess - setup the entity
func setupEntityCacheAccess(store datastore.Store) {
	cacheAccessEntityMetadata = datastore.MetadataProvider()
	cacheAccessEntityMetadata.Name = "ca"
	cacheAccessEntityMetadata.DB = "MetaRecordDB"
	cacheAccessEntityMetadata.Provider = ProviderCacheAccess
	cacheAccessEntityMetadata.Store = store

	datastore.RegisterEntityMetadata("ca", cacheAccessEntityMetadata)
	cacheAccessEntityCollection = &datastore.EntityCollection{
		CollectionName:     "collection." + sortedSetCacheAccessTimeHash,
		CollectionSize:     60000000,
		CollectionDuration: time.Hour,
	}
}

var cacheAccessEntityCollection *datastore.EntityCollection
