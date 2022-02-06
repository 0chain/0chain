package blockstore

import "github.com/go-redis/redis"

var (
	bwrecDB blockStorage
)

type (
	blockStorage interface {
		Delete(key string) error
		DeleteFromHash(tablaBName, key string) error
		DeleteSorted(key, value string) error
		Exec() error
		Get(key string) ([]byte, error)
		GetCountSorted(tablaBName string) (int64, error)
		GetFromHash(key, value string) ([]interface{}, error)
		GetSortedRange(key string, start, stop int64) ([]string, error)
		GetSortedRangeByScore(key string, lastBlock, count int64) []*UnmovedBlockRecord
		Set(key string, value []byte) error
		SetSorted(key string, score float64, value string) error
		SetToHash(tablaBName, key, value string) error
		StartTx() error
	}
)

func newBlockStore(deleteExistingDB bool) {
	bwrecDB = newBwrDB(deleteExistingDB)

}

func newBwrDB(deleteExistingDB bool) blockStorage {
	db := blockStore{}
	db.Client = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	if deleteExistingDB {
		_, _ = db.Client.FlushDB().Result()
	}

	return &db
}
