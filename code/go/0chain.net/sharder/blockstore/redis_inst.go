package blockstore

import (
	"fmt"
	"github.com/go-redis/redis"
	"time"
)

type (
	blockStore struct {
		Client *redis.Client
	}
)

func (db *blockStore) Delete(key string) error {
	return db.Client.Del(key).Err()
}

func (db *blockStore) DeleteFromHash(tableBName, key string) error {
	return db.Client.HDel(tableBName, key).Err()
}

func (db *blockStore) DeleteSorted(key, value string) error {
	return db.Client.ZRem(key, value).Err()
}

func (db *blockStore) Exec() error {
	return db.Client.Do("exec").Err()
}

func (db *blockStore) Get(key string) ([]byte, error) {
	return db.Client.Get(key).Bytes()
}

func (db *blockStore) GetCountSorted(tableBName string) (int64, error) {
	return db.Client.ZCount(tableBName, "-inf", "+inf").Result()
}

func (db *blockStore) GetFromHash(key, value string) ([]interface{}, error) {
	return db.Client.HMGet(key, value).Result()
}

func (db *blockStore) GetSortedRange(key string, start, stop int64) ([]string, error) {
	return db.Client.ZRange(key, start, stop).Result()
}

func (db *blockStore) GetSortedRangeByScore(key string, lastBlock, count int64) []*UnmovedBlockRecord {
	ubrsZ, _ := db.Client.ZRangeByScoreWithScores(
		key,
		redis.ZRangeBy{Min: "-inf", Max: fmt.Sprintf("%v", lastBlock), Offset: 0, Count: count},
	).Result()

	if ubrsZ == nil {
		return nil
	}
	var ubrs []*UnmovedBlockRecord
	for _, ubr := range ubrsZ {
		ubrs = append(ubrs, &UnmovedBlockRecord{CreatedAt: time.UnixMicro(int64(ubr.Score)), Hash: ubr.Member.(string)})
	}

	return ubrs
}

func (db *blockStore) Set(key string, value []byte) error {
	status := db.Client.Set(key, value, 0)
	if err := status.Err(); err != nil {
		return err
	}

	return nil
}

func (db *blockStore) SetSorted(key string, score float64, value string) error {
	return db.Client.ZAdd(key, redis.Z{Member: value, Score: score}).Err()
}

func (db *blockStore) SetToHash(tableBName, key, value string) error {
	m := map[string]interface{}{}
	m[key] = value
	return db.Client.HMSet(tableBName, m).Err()
}

func (db *blockStore) StartTx() error {
	return db.Client.Do("multi").Err()
}
