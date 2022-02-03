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

// Delete remove the specified member from the set stored at key.
func (db *blockStore) Delete(key string) error {
	return db.Client.Del(key).Err()
}

// DeleteFromHash delete hash field.
func (db *blockStore) DeleteFromHash(tableBName, key string) error {
	return db.Client.HDel(tableBName, key).Err()
}

// DeleteFromSorted remove member from a sorted set.
func (db *blockStore) DeleteFromSorted(tableBName, key string) error {
	return db.Client.ZRem(tableBName, key).Err()
}

// Exec execute all commands issued after MULTI.
func (db *blockStore) Exec() error {
	return db.Client.Do("exec").Err()
}

// Get gets the value of the key.
func (db *blockStore) Get(key string) ([]byte, error) {
	return db.Client.Get(key).Bytes()
}

// GetCountFromSorted count the members in a sorted set with scores within the given values.
func (db *blockStore) GetCountFromSorted(tableBName string) (int64, error) {
	return db.Client.ZCount(tableBName, "-inf", "+inf").Result()
}

// GetFromHash get the value of a hash field.
func (db *blockStore) GetFromHash(tableBName, key string) (interface{}, error) {
	return db.Client.HGet(tableBName, key).Result()
}

// GetRangeFromSorted return a range of members in a sorted set.
func (db *blockStore) GetRangeFromSorted(key string, start, stop int64) ([]string, error) {
	return db.Client.ZRange(key, start, stop).Result()
}

// GetRangeByScoreFromSorted return a range of members in a sorted set, by score.
func (db *blockStore) GetRangeByScoreFromSorted(key string, lastBlock, count int64) []*UnmovedBlockRecord {
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

// Set add the specified member to the set stored at key.
func (db *blockStore) Set(key string, value []byte) error {
	return db.Client.Set(key, value, 0).Err()
}

// SetToSorted member to a sorted set, or update its score if it already exists.
func (db *blockStore) SetToSorted(key string, score float64, value string) error {
	return db.Client.ZAdd(key, redis.Z{Member: value, Score: score}).Err()
}

// SetToHash set the string value of a hash field.
func (db *blockStore) SetToHash(tableBName, key, value string) error {
	return db.Client.HSet(tableBName, key, value).Err()
}

// StartTx mark the start of a transaction block.
func (db *blockStore) StartTx() error {
	return db.Client.Do("multi").Err()
}
