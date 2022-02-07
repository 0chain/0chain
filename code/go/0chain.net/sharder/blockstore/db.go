package blockstore

import (
	"encoding/json"
	"fmt"
	"github.com/go-redis/redis"
	"strings"
	"time"
)

type WhichTier uint8

//db variables
var (
	bwrRedis  blockStore
	startTime = time.Date(1970, 1, 1, 0, 0, 0, 0, time.Local)
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

// redis constant values
const (
	DefaultHostRedisDB     = "localhost"
	DefaultPortRedisDB     = "6379"
	DefaultPasswordRedisDB = ""
	DefaultNumberRedisDB   = 0

	redisHashCacheHashAccessTime      = "redisHashCacheHashAccessTime"
	redisSortedSetCacheAccessTimeHash = "redisSortedSetCacheAccessTimeHash"
	redisSortedSetUnmovedBlock        = "redisSortedSetUnmovedBlock"

	CacheAccessTimeSeparator = ":"
)

// InitMetaRecordDB Create db file and create buckets.
func InitMetaRecordDB(host, port, password string, numDB int, deleteExistingDB bool) {

	bwrRedis.Client = redis.NewClient(&redis.Options{
		Addr:     host + ":" + port,
		Password: password,
		DB:       numDB,
	})

	if deleteExistingDB {
		_, _ = bwrRedis.Client.FlushDB().Result()
	}
}

// BlockWhereRecord It simply provides whereabouts of a block. It can be in Warm Tier, Cold Tier, Hot and Warm Tier, Hot and Cold Tier, etc.
type BlockWhereRecord struct {
	Hash      string    `json:"-"`
	Tiering   WhichTier `json:"tr"`
	BlockPath string    `json:"vp,omitempty"` //For disk volume it is simple unix path. For cold storage it is "storageUrl:bucketName".
	ColdPath  string    `json:"cp,omitempty"`
}

// AddOrUpdate Add or Update whereabout of a block.
func (bwr *BlockWhereRecord) AddOrUpdate() (err error) {
	value, err := json.Marshal(bwr)
	if err != nil {
		return err
	}

	return bwrRedis.Set(bwr.Hash, value)
}

// GetBlockWhereRecord Get whereabout of a block.
func GetBlockWhereRecord(hash string) (*BlockWhereRecord, error) {
	data, err := bwrRedis.Get(hash)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, fmt.Errorf("Block meta record for %v not found.", hash)
	}

	bwr := BlockWhereRecord{}
	err = json.Unmarshal(data, &bwr)
	if err != nil {
		return nil, err
	}

	bwr.Hash = hash
	return &bwr, nil
}

// DeleteBlockWhereRecord Delete metadata.
func DeleteBlockWhereRecord(hash string) error {
	return bwrRedis.Delete(hash)
}

// UnmovedBlockRecord Unmoved blocks; If cold tiering is enabled then record of unmoved blocks will be kept inside UnmovedBlockRecord bucket.
//Some worker will query for the unmoved block and if it is within the date range then it will be moved to the cold storage.
type UnmovedBlockRecord struct {
	CreatedAt time.Time `json:"crAt"`
	Hash      string    `json:"hs"`
}

func (ubr *UnmovedBlockRecord) Add() (err error) {
	ubr.CreatedAt = ubr.CreatedAt.Truncate(time.Microsecond)
	endTime := time.Date(
		ubr.CreatedAt.Year(),
		ubr.CreatedAt.Month(),
		ubr.CreatedAt.Day(),
		ubr.CreatedAt.Hour(),
		ubr.CreatedAt.Minute(),
		ubr.CreatedAt.Second(),
		ubr.CreatedAt.Nanosecond(),
		time.Local,
	)
	difference := endTime.Sub(startTime)

	return bwrRedis.SetToSorted(redisSortedSetUnmovedBlock, float64(difference.Microseconds()), ubr.Hash)
}

func (ubr *UnmovedBlockRecord) Delete() (err error) {
	return bwrRedis.DeleteFromSorted(redisSortedSetUnmovedBlock, ubr.Hash)
}

// GetUnmovedBlocks returns the number of blocks = count from the range [0,lastBlock).
func GetUnmovedBlocks(lastBlock, count int64) []*UnmovedBlockRecord {
	ubrsZ, _ := bwrRedis.GetRangeByScoreFromSorted(redisSortedSetUnmovedBlock, lastBlock, count)
	var ubrs []*UnmovedBlockRecord
	for _, ubr := range ubrsZ {
		t := time.Duration(int64(ubr.Score))
		ubrs = append(
			ubrs,
			&UnmovedBlockRecord{CreatedAt: startTime.Add(t * time.Microsecond), Hash: ubr.Member.(string)},
		)
	}

	return ubrs
}

//Add a cache bucket to store accessed time as key and hash as its value
//eg accessedTime:hash
//Use sorting feature of boltdb to quickly delete cached blocks that should be replaced
type cacheAccess struct {
	Hash       string
	AccessTime *time.Time
}

func GetHashKeysForReplacement() chan *cacheAccess {
	ch := make(chan *cacheAccess, 10)

	go func() {
		defer func() {
			close(ch)
		}()

		i, _ := bwrRedis.GetCountFromSorted(redisSortedSetCacheAccessTimeHash)
		i /= 2 //Number of blocks to replace
		var startRange int64 = 0
		var endRange int64 = 0
		k := int(i)
		for j := 0; j < k; j = int(endRange) {
			endRange += 1000
			if endRange > i {
				endRange = i
			}
			blocks, _ := bwrRedis.GetRangeFromSorted(redisSortedSetCacheAccessTimeHash, startRange, endRange)
			for _, block := range blocks {
				ca := new(cacheAccess)
				sl := strings.Split(block, CacheAccessTimeSeparator)
				ca.Hash = sl[1]
				accessTime, _ := time.Parse(time.RFC3339, sl[0])
				ca.AccessTime = &accessTime
				ch <- ca
			}
			startRange = endRange + 1
		}
	}()

	return ch
}

func (ca *cacheAccess) addOrUpdate() error {
	timeStr := ca.AccessTime.Format(time.RFC3339)
	accessTimeKey := fmt.Sprintf("%v%v%v", timeStr, CacheAccessTimeSeparator, ca.Hash)

	timeValue, err := bwrRedis.GetFromHash(redisHashCacheHashAccessTime, ca.Hash)
	if err != nil {
		return err
	}

	if bwrRedis.StartTx() != nil {
		return err
	}
	if timeValue != nil {
		delKey := fmt.Sprintf("%v%v%v", timeValue.(string), CacheAccessTimeSeparator, ca.Hash)
		err = bwrRedis.DeleteFromSorted(redisSortedSetCacheAccessTimeHash, delKey)
	}
	err = bwrRedis.SetToSorted(redisSortedSetCacheAccessTimeHash, 0.0, accessTimeKey)
	err = bwrRedis.SetToHash(redisHashCacheHashAccessTime, ca.Hash, timeStr)

	err = bwrRedis.Exec()
	if err != nil {
		return err
	}

	return nil
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
	err := bwrRedis.StartTx()
	if err != nil {
		return err
	}
	err = bwrRedis.DeleteFromSorted(redisSortedSetCacheAccessTimeHash,
		fmt.Sprintf("%v%v%v", ca.AccessTime.Format(time.RFC3339), CacheAccessTimeSeparator, ca.Hash),
	)
	err = bwrRedis.DeleteFromHash(redisHashCacheHashAccessTime, ca.Hash)

	err = bwrRedis.Exec()
	if err != nil {
		return err
	}

	return nil
}
