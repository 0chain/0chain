package smartblockstore

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
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
	WarmTier             WhichTier = 2 //Warm tier only
	HotTier              WhichTier = 4 //Hot tier only
	ColdTier             WhichTier = 8 //Cold tier only
	CacheAndHotTier      WhichTier = 5
	HotAndColdTier       WhichTier = 12
	CacheHotAndColdTier  WhichTier = 13
	WarmAndColdTier      WhichTier = 10
	CacheWarmAndColdTier WhichTier = 11
	CacheAndWarmTier     WhichTier = 3
	CacheAndColdTier     WhichTier = 9
)

//bucket constant values
const (
	BlockWhereBucket   = "bwb"
	UnmovedBlockBucket = "ubb"
	BlockUsageBucket   = "bub"
)

//Create db file and create buckets
func InitMetaRecordDB(deleteExistingDB bool) {
	wg := sync.WaitGroup{}
	wg.Add(2)

	if deleteExistingDB {
		os.Remove("path/to/db")
		os.Remove("path/to/other/db")
	}
	//Open db for storing whereabout of blocks
	go func() {
		defer wg.Done()
		var err error
		bwrDB, err = bbolt.Open("path/to/db", 0644, bbolt.DefaultOptions)
		if err != nil {
			panic(err)
		}

		err = bwrDB.Update(func(t *bbolt.Tx) error {
			_, err := t.CreateBucketIfNotExists([]byte(BlockUsageBucket))
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
		qDB, err = bbolt.Open("path/to/db", 0644, bbolt.DefaultOptions)
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
	CachePath string    `json:"chp,omitempty"`
	CloudPath string    `json:"cp,omitempty"`
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
	unixTime := ubr.CreatedAt.UnixNano()
	key := []byte(strconv.FormatInt(unixTime, 10))

	return bwrDB.Update(func(t *bbolt.Tx) error {
		bkt := t.Bucket([]byte(UnmovedBlockBucket))
		return bkt.Delete(key)
	})
}

func GetUnmovedBlock(prevKey, upto []byte) (ubr *UnmovedBlockRecord, timeByte []byte, err error) {
	var hashByte []byte
	err = bwrDB.View(func(t *bbolt.Tx) error {
		cursor := t.Bucket([]byte(UnmovedBlockBucket)).Cursor()
		k, v := cursor.Seek(prevKey)

		if bytes.Compare(k, prevKey) == 0 {
			k, v = cursor.Next()
		}

		if k == nil || bytes.Compare(k, upto) > 0 {
			return nil
		} else {
			timeByte = make([]byte, len(k))
			hashByte = make([]byte, len(v))
			copy(timeByte, k)
			copy(hashByte, v)
		}

		return nil
	})

	if err != nil || timeByte == nil {
		return
	}

	createdAt, _ := time.Parse(time.RFC3339, string(timeByte))

	ubr = &UnmovedBlockRecord{
		CreatedAt: createdAt,
		Hash:      string(hashByte),
	}

	return
}
