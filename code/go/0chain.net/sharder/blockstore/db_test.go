package blockstore

import (
	"fmt"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	redisClient = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	code := m.Run()
	os.Exit(code)
}

func Test_BwrAddOrUpdate(t *testing.T) {
	redisClient.FlushDB()
	bwr := mockBWR()

	tests := []struct {
		name      string
		bwr       *BlockWhereRecord
		wantBlock *BlockWhereRecord
	}{
		{
			name:      "OK",
			bwr:       bwr,
			wantBlock: bwr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b *BlockWhereRecord
			got, _ := GetBlockWhereRecord(test.bwr.Hash)
			assert.Equal(t, b, got)
			err := test.bwr.AddOrUpdate()
			assert.NoError(t, err)
			got, err = GetBlockWhereRecord(test.bwr.Hash)
			assert.NoError(t, err)
			assert.Equal(t, test.wantBlock, got)
		})
	}
}

func Test_DeleteBlockWhereRecord(t *testing.T) {
	redisClient.FlushDB()
	bwr := mockBWR()

	tests := []struct {
		name      string
		bwr       *BlockWhereRecord
		wantBlock *BlockWhereRecord
		wantError bool
	}{
		{
			name:      "OK",
			bwr:       bwr,
			wantBlock: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.bwr.AddOrUpdate()
			assert.NoError(t, err)
			got, _ := GetBlockWhereRecord(test.bwr.Hash)
			assert.Equal(t, test.bwr, got)
			err = DeleteBlockWhereRecord(test.bwr.Hash)
			assert.NoError(t, err)
			got, _ = GetBlockWhereRecord(test.bwr.Hash)
			assert.Equal(t, test.wantBlock, got)
		})
	}
}

func Test_GetBlockWhereRecord(t *testing.T) {
	redisClient.FlushDB()
	bwr := mockBWR()

	tests := []struct {
		name      string
		bwr       *BlockWhereRecord
		wantBlock *BlockWhereRecord
	}{
		{
			name:      "OK",
			bwr:       bwr,
			wantBlock: bwr,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.bwr.AddOrUpdate()
			assert.NoError(t, err)
			got, err := GetBlockWhereRecord(test.bwr.Hash)
			assert.NoError(t, err)
			assert.Equal(t, test.wantBlock, got)
		})
	}
}

func Test_AddUnmovedBlockRecord(t *testing.T) {
	redisClient.FlushDB()
	ubr := mockUBR()
	n := time.Now()
	endTime := time.Date(
		n.Year(),
		n.Month(),
		n.Day(),
		n.Hour(),
		n.Minute(),
		n.Second(),
		n.Nanosecond(),
		time.Local,
	)
	difference := endTime.Sub(startTime).Microseconds()

	tests := []struct {
		name       string
		lastBlock  int64
		ubr        *UnmovedBlockRecord
		wantBlocks []*UnmovedBlockRecord
	}{
		{
			name:       "OK",
			lastBlock:  difference,
			ubr:        ubr,
			wantBlocks: []*UnmovedBlockRecord{ubr},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var b []*UnmovedBlockRecord
			blocks := GetUnmovedBlocks(test.lastBlock, 1)
			assert.Equal(t, b, blocks)
			err := test.ubr.Add()
			assert.NoError(t, err)
			blocks = GetUnmovedBlocks(test.lastBlock, 1)
			assert.Equal(t, test.wantBlocks, blocks)
		})
	}
}

func Test_DeleteUnmovedBlockRecord(t *testing.T) {
	redisClient.FlushDB()
	ubr := mockUBR()
	n := time.Now()
	endTime := time.Date(
		n.Year(),
		n.Month(),
		n.Day(),
		n.Hour(),
		n.Minute(),
		n.Second(),
		n.Nanosecond(),
		time.Local,
	)
	difference := endTime.Sub(startTime).Microseconds()

	tests := []struct {
		name       string
		lastBlock  int64
		ubr        *UnmovedBlockRecord
		wantBlocks []*UnmovedBlockRecord
	}{
		{
			name:       "OK",
			lastBlock:  difference,
			ubr:        ubr,
			wantBlocks: []*UnmovedBlockRecord{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.ubr.Add()
			assert.NoError(t, err)
			blocks := GetUnmovedBlocks(test.lastBlock, 1)
			assert.NotEqual(t, test.wantBlocks, blocks)
			err = test.ubr.Delete()
			assert.NoError(t, err)
			blocks = GetUnmovedBlocks(test.lastBlock, 1)
			if blocks != nil {
				t.Errorf("Delete want %v | got %v", test.wantBlocks, blocks)
			}
		})
	}
}

func Test_GetUnmovedBlocks(t *testing.T) {
	redisClient.FlushDB()
	var ubrs []*UnmovedBlockRecord
	var count = 5
	for i := 0; i < count; i++ {
		u := mockUBR()
		err := u.Add()
		if err != nil {
			t.Fatalf("unable to create test structure")
		}
		ubrs = append(ubrs, u)
	}
	n := time.Now()
	endTime := time.Date(
		n.Year(),
		n.Month(),
		n.Day(),
		n.Hour(),
		n.Minute(),
		n.Second(),
		n.Nanosecond(),
		time.Local,
	)
	difference := endTime.Sub(startTime).Microseconds()

	tests := []struct {
		name      string
		lastBlock int64
		count     int64
		want      []*UnmovedBlockRecord
	}{
		{
			name:      "OK",
			lastBlock: difference,
			count:     int64(count),
			want:      ubrs,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := GetUnmovedBlocks(test.lastBlock, test.count)
			assert.Equal(t, test.want, got)
		})
	}
}

func Test_addOrUpdate(t *testing.T) {
	redisClient.FlushDB()
	ca := mockCacheAccess()

	tests := []struct {
		name         string
		cacheAccess  *cacheAccess
		wantTime     string
		wantTimeHash []string
	}{
		{
			name:         "OK",
			cacheAccess:  ca,
			wantTime:     ca.AccessTime.Format(time.RFC3339Nano),
			wantTimeHash: []string{fmt.Sprintf("%v%v%v", ca.AccessTime.Format(time.RFC3339Nano), CacheAccessTimeSeparator, ca.Hash)},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cacheAccess.addOrUpdate()
			assert.Error(t, err)
			gotTime, _ := redisClient.HGet(redisHashCacheHashAccessTime, test.cacheAccess.Hash).Result()
			assert.Equal(t, test.wantTime, gotTime)
			gotTimeHash, _ := redisClient.ZRange(redisSortedSetCacheAccessTimeHash, 0, 1).Result()
			assert.Equal(t, test.wantTimeHash, gotTimeHash)
		})
	}
}

func Test_delete(t *testing.T) {
	redisClient.FlushDB()
	ca := mockCacheAccess()

	tests := []struct {
		name         string
		cacheAccess  *cacheAccess
		wantTime     string
		wantTimeHash []string
	}{
		{
			name:         "OK",
			cacheAccess:  ca,
			wantTime:     "",
			wantTimeHash: []string{},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := test.cacheAccess.addOrUpdate()
			assert.Error(t, err, nil)
			gotTime, _ := redisClient.HGet(redisHashCacheHashAccessTime, test.cacheAccess.Hash).Result()
			assert.Equal(t, test.cacheAccess.AccessTime.Format(time.RFC3339Nano), gotTime)
			gotTimeHash, _ := redisClient.ZRange(redisSortedSetCacheAccessTimeHash, 0, 1).Result()
			assert.Equal(t, []string{fmt.Sprintf("%v%v%v", ca.AccessTime.Format(time.RFC3339Nano), CacheAccessTimeSeparator, ca.Hash)}, gotTimeHash)
			err = test.cacheAccess.delete()
			assert.NoError(t, err)
			gotTime, _ = redisClient.HGet(redisHashCacheHashAccessTime, test.cacheAccess.Hash).Result()
			assert.Equal(t, test.wantTime, gotTime)
			gotTimeHash, _ = redisClient.ZRange(redisSortedSetCacheAccessTimeHash, 0, 1).Result()
			assert.Equal(t, test.wantTimeHash, gotTimeHash)
		})
	}
}

func Test_GetHashKeysForReplacement(t *testing.T) {
	redisClient.FlushDB()
	var cas []*cacheAccess
	var count = 10
	for i := 0; i < count; i++ {
		ca := mockCacheAccess()
		err := ca.addOrUpdate()
		if err.Error() != "redis: nil" {
			t.Fatalf("unable to create test structure")
		}
		if i <= (count / 2) {
			cas = append(cas, ca)
		}
	}

	tests := []struct {
		name       string
		wantBlocks []*cacheAccess
	}{
		{
			name:       "OK",
			wantBlocks: cas,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var got []*cacheAccess
			for ca := range GetHashKeysForReplacement() {
				got = append(got, ca)
			}
			assert.Equal(t, test.wantBlocks, got)
		})
	}
}
