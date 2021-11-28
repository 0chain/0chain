package blockstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.etcd.io/bbolt"
)

type WhichTier uint8

//db variables
var (
	bwrDB *bbolt.DB
	qDB   *bbolt.DB //query db
)

/*
Cache = 1
Warm  = 2
Hot   = 4
Cold  = 8
*/
const (
	WarmTier        WhichTier = 2 //Warm tier only
	HotTier         WhichTier = 4 //Hot tier only
	ColdTier        WhichTier = 8 //Cold tier only
	HotAndColdTier  WhichTier = 12
	WarmAndColdTier WhichTier = 10
)

//bucket constant values
const (
	DefaultBlockMetaRecordDB = "/meta/bmr.db"
	DefaultQueryMetaRecordDB = "/meta/qmr.db"
	BlockWhereBucket         = "bwb"
	UnmovedBlockBucket       = "ubb"
	BlockUsageBucket         = "bub"
	//Contains key that is combination of "accessTime:hash" and value of nil
	CacheAccessTimeHashBucket = "cahb"
	CacheAccessTimeSeparator  = ":"
	//Contains key value; "hash:accessTime"
	CacheHashAccessTimeBucket = "chab"
)

//Create db file and create buckets
func InitMetaRecordDB(bmrDB, qmrDB string, deleteExistingDB bool) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	if deleteExistingDB {
		os.Remove(bmrDB)
		os.Remove(qmrDB)
	}
	//Open db for storing whereabout of blocks
	go func() {
		defer wg.Done()
		var err error
		parentDir, _ := filepath.Split(bmrDB)
		if err := os.MkdirAll(parentDir, 0644); err != nil {
			panic(err)
		}

		bwrDB, err = bbolt.Open(bmrDB, 0644, bbolt.DefaultOptions) // fix me
		if err != nil {
			panic(err)
		}

		err = bwrDB.Update(func(t *bbolt.Tx) error {
			_, err := t.CreateBucketIfNotExists([]byte(BlockWhereBucket))
			if err != nil {
				return err
			}
			_, err = t.CreateBucketIfNotExists([]byte(UnmovedBlockBucket))
			return err
		})

		if err != nil {
			panic(err)
		}

	}()

	go func() {
		defer wg.Done()
		var err error
		parentDir, _ := filepath.Split(qmrDB)
		if err := os.MkdirAll(parentDir, 0644); err != nil {
			panic(err)
		}

		qDB, err = bbolt.Open(qmrDB, 0644, bbolt.DefaultOptions) // fix me
		if err != nil {
			panic(err)
		}

		err = qDB.Update(func(t *bbolt.Tx) error {
			_, err = t.CreateBucketIfNotExists([]byte(BlockUsageBucket))
			return err
		})
	}()

	wg.Wait()
}

//It simply provides whereabouts of a block. It can be in Warm Tier, Cold Tier, Hot and Warm Tier, Hot and Cold Tier, etc.
type BlockWhereRecord struct {
	Hash      string    `json:"-"`
	Tiering   WhichTier `json:"tr"`
	BlockPath string    `json:"vp,omitempty"` //For disk volume it is simple unix path. For cold storage it is "storageUrl:bucketName"
	ColdPath  string    `json:"cp,omitempty"`
}

//Add or Update whereabout of a block
func (bwr *BlockWhereRecord) AddOrUpdate() (err error) {
	value, err := json.Marshal(bwr)

	if err != nil {
		return err
	}
	key := []byte(bwr.Hash)

	err = bwrDB.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BlockWhereBucket))
		return bkt.Put(key, value)
	})

	return
}

//Get whereabout of a block
func GetBlockWhereRecord(hash string) (*BlockWhereRecord, error) {
	var data []byte
	var bwr BlockWhereRecord
	key := []byte(hash)

	err := bwrDB.View(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BlockWhereBucket))
		bwrData := bkt.Get(key)

		if bwrData == nil {
			return fmt.Errorf("Block meta record for %v not found.", hash)
		}

		data = make([]byte, len(bwrData))
		copy(data, bwrData)
		return nil
	})

	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(data, &bwr)
	if err != nil {
		return nil, err
	}

	bwr.Hash = hash
	return &bwr, nil
}

//Delete metadata
func DeleteBlockWhereRecord(hash string) {
	bwrDB.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(BlockWhereBucket))
		if bkt == nil {
			return nil
		}

		return bkt.Delete([]byte(hash))
	})
}

//Unmoved blocks; If cold tiering is enabled then record of unmoved blocks will be kept inside UnmovedBlockRecord bucket.
//Some worker will query for the unmoved block and if it is within the date range then it will be moved to the cold storage.
type UnmovedBlockRecord struct {
	CreatedAt time.Time `json:"crAt"`
	Hash      string    `json:"hs"`
}

func (ubr *UnmovedBlockRecord) Add() (err error) {
	key := []byte(ubr.CreatedAt.Format(time.RFC3339))
	value := []byte(ubr.Hash)

	err = bwrDB.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(UnmovedBlockBucket))
		return bkt.Put(key, value)
	})

	return
}

func (ubr *UnmovedBlockRecord) Delete() (err error) {
	key := []byte(ubr.CreatedAt.Format(time.RFC3339))
	return bwrDB.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(UnmovedBlockBucket))
		return bkt.Delete(key)
	})
}

func GetUnmovedBlock(prevKey, upto []byte) (ubr *UnmovedBlockRecord, timeByte []byte) {
	var hashByte []byte
	bwrDB.View(func(t *bbolt.Tx) error {
		cursor := t.Bucket([]byte(UnmovedBlockBucket)).Cursor()
		k, v := cursor.Seek(prevKey)

		if bytes.Compare(k, prevKey) == 0 {
			k, v = cursor.Next()
		}

		if k == nil || bytes.Compare(k, upto) > 0 {
			return nil
		}

		timeByte = make([]byte, len(k))
		hashByte = make([]byte, len(v))
		copy(timeByte, k)
		copy(hashByte, v)

		return nil
	})

	if timeByte == nil {
		return
	}

	createdAt, _ := time.Parse(time.RFC3339, string(timeByte))

	ubr = &UnmovedBlockRecord{
		CreatedAt: createdAt,
		Hash:      string(hashByte),
	}

	return
}

//Add a cache bucket to store accessed time as key and hash as its value
//eg accessedTime:hash
//Use sorting feature of boltdb to quickly delete cached blocks that should be replaced
type cacheAccess struct {
	Hash       string
	AccessTime *time.Time
}

func GetHashKeyForReplacement() chan *cacheAccess {
	ch := make(chan *cacheAccess, 10)

	go func() {
		defer func() {
			close(ch)
		}()

		bwrDB.View(func(t *bbolt.Tx) error {
			bkt := t.Bucket([]byte(CacheAccessTimeHashBucket))
			if bkt == nil {
				return nil
			}

			i := bkt.Stats().KeyN / 2 //Number of blocks to replace
			count := 0

			cursor := bkt.Cursor()
			for k, _ := cursor.Next(); k != nil && count < i; k, _ = cursor.Next() {
				ca := new(cacheAccess)
				sl := strings.Split(string(k), CacheAccessTimeSeparator)
				ca.Hash = sl[1]
				accessTime, _ := time.Parse(time.RFC3339, sl[0])
				ca.AccessTime = &accessTime
				ch <- ca
				count++
			}
			return nil
		})

	}()

	return ch
}

func (ca *cacheAccess) addOrUpdate() error {
	timeStr := ca.AccessTime.Format(time.RFC3339)
	accessTimeKey := []byte(fmt.Sprintf("%v:%v", timeStr, ca.Hash))

	return bwrDB.Update(func(t *bbolt.Tx) error {
		accessTimeBkt := t.Bucket([]byte(CacheAccessTimeHashBucket))
		if accessTimeBkt == nil {
			return fmt.Errorf("%v bucket does not exist", CacheAccessTimeHashBucket)
		}

		hashBkt := t.Bucket([]byte(CacheHashAccessTimeBucket))
		if hashBkt == nil {
			return fmt.Errorf("%v bucket does not exist", CacheHashAccessTimeBucket)
		}

		timeValue := hashBkt.Get([]byte(ca.Hash))
		if timeValue != nil {
			delKey := []byte(fmt.Sprintf("%v%v%v", string(timeValue), CacheAccessTimeSeparator, ca.Hash))
			accessTimeBkt.Delete(delKey)
		}

		if err := accessTimeBkt.Put(accessTimeKey, nil); err != nil {
			return err
		}

		return hashBkt.Put([]byte(ca.Hash), []byte(timeStr))
	})
}

// func (ca *cacheAccess) update() {
// 	timeStr := ca.AccessTime.Format(time.RFC3339)
// 	accessTimeKey := []byte(fmt.Sprintf("%v%v%v", timeStr, CacheAccessTimeSeparator, ca.Hash))

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
// 			k := []byte(fmt.Sprintf("%v%v%v", string(oldAccessTime), CacheAccessTimeSeparator, ca.Hash))
// 			accessTimeBkt.Delete(k)
// 		}

// 		if err := hashBkt.Put([]byte(ca.Hash), []byte(timeStr)); err != nil {
// 			return err
// 		}
// 		return accessTimeBkt.Put(accessTimeKey, nil)
// 	})
// }

func (ca *cacheAccess) delete() error {
	return bwrDB.Update(func(t *bbolt.Tx) error {
		tBucket := t.Bucket([]byte(CacheAccessTimeHashBucket))
		if tBucket == nil {
			return nil
		}

		hBucket := t.Bucket([]byte(CacheHashAccessTimeBucket))
		if hBucket == nil {
			return nil
		}

		tKey := []byte(fmt.Sprintf("%v%v%v", ca.AccessTime.Format(time.RFC3339), CacheAccessTimeSeparator, ca.Hash))
		if err := tBucket.Delete(tKey); err != nil {
			return err
		}

		return hBucket.Delete([]byte(ca.Hash))
	})
}
