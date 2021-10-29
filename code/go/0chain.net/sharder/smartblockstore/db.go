package smartblockstore

import (
	"bytes"
	"encoding/json"
	"fmt"
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

//bucket constant values
const (
	BlockWhereBucket   = "bwb"
	UnmovedBlockBucket = "ubb"
	BlockUsageBucket   = "bub"

	HotTier          WhichTier = iota //Hot tier only
	WarmTier                          //Warm tier only
	ColdTier                          //Cold tier only
	CacheAndWarmTier                  //Cache and warm tier
	CacheAndColdTier                  //Cache and cold tier
)

//Create db file and create buckets
func InitMetaRecordDB() {
	wg := sync.WaitGroup{}
	wg.Add(2)

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
	hash      string    `json:"-"`
	tiering   WhichTier `json:"tr"`
	blockPath string    `json:"vp,omitempty"`
	cachePath string    `json:"cp,omitempty"`
}

//Add or Update whereabout of a block
func (bwr *BlockWhereRecord) AddOrUpdate() (err error) {
	value, err := json.Marshal(bwr)
	if err != nil {
		return err
	}
	key := []byte(bwr.hash)

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
	bwr.hash = hash
	return &bwr, nil
}

//Unmoved blocks; If cold tiering is enabled then record of unmoved blocks will be kept inside UnmovedBlockRecord bucket.
//Some worker will query for the unmoved block and if it is within the date range then it will be moved to the cold storage.
type UnmovedBlockRecord struct {
	CreatedAt time.Time `json:"crAt"`
	Hash      string    `json:"hs"`
}

func (ubr *UnmovedBlockRecord) Add() (err error) {
	unixTime := ubr.CreatedAt.UnixNano()
	key := []byte(strconv.FormatInt(unixTime, 10))
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

func GetUnmovedBlocks(prevKey, minPrefix, maxPrefix string) (ubr *UnmovedBlockRecord, err error) {
	prev := []byte(prevKey)
	min := []byte(minPrefix)
	max := []byte(maxPrefix)

	var hashByte, timeByte []byte
	err = qDB.View(func(t *bbolt.Tx) error {
		cursor := t.Bucket([]byte(UnmovedBlockBucket)).Cursor()
		k, v := cursor.Seek(min)
		if bytes.Compare(k, prev) == 0 {
			k, v = cursor.Next()
		}
		if bytes.Compare(max, k) <= 0 {
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
	i, _ := strconv.ParseInt(string(timeByte), 10, 64)
	createdAt := time.Unix(0, i)
	ubr = &UnmovedBlockRecord{
		CreatedAt: createdAt,
		Hash:      string(hashByte),
	}
	return
}
