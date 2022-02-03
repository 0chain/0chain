package blockstore

import (
	"encoding/json"
	"github.com/alicebob/miniredis/v2"
	"github.com/go-redis/redis"
	"github.com/stretchr/testify/assert"
	"log"
	"os"
	"reflect"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	mr, err := miniredis.Run()
	if err != nil {
		log.Fatalf("an error '%s' was not expected when opening a stub database connection", err)
	}

	bwrRedis.Client = redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	code := m.Run()
	os.Exit(code)
}

func Test_Delete(t *testing.T) {
	bwrRedis.Client.FlushDB()
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want []byte
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: make([]byte, 0),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bwrRedis.Set(test.ubr.Hash, nil)
			if err != nil {
				t.Fatal(err)
			}
			err = bwrRedis.Delete(test.ubr.Hash)
			assert.NoError(t, err)
			got, _ := bwrRedis.Get(test.ubr.Hash)
			assert.Equal(t, test.want, got)
		})
	}
}

func Test_Get(t *testing.T) {
	bwrRedis.Client.FlushDB()
	ubr := mockBWR()
	val, err := json.Marshal(&ubr)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want []byte
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: val,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bwrRedis.Set(test.ubr.Hash, test.want)
			assert.NoError(t, err)
			got, _ := bwrRedis.Get(test.ubr.Hash)
			assert.Equal(t, test.want, got)
		})
	}
}

func Test_Set(t *testing.T) {
	bwrRedis.Client.FlushDB()
	ubr := mockUBR()
	val, err := json.Marshal(&ubr)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want []byte
	}{
		{
			name: "OK",
			ubr:  ubr,
			want: val,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			blob, err := json.Marshal(&test.ubr)
			if err != nil {
				t.Fatal(err)
			}
			err = bwrRedis.Set(test.ubr.Hash, blob)
			assert.NoError(t, err)
			got, err := bwrRedis.Get(test.ubr.Hash)
			if err != nil {
				t.Fatal(err)
			}
			assert.Equal(t, val, got)
		})
	}
}

func Test_DeleteFromHash(t *testing.T) {
	bwrRedis.Client.FlushDB()
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want string
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bwrRedis.SetToHash(redisHashCacheHashAccessTime, test.ubr.Hash, test.ubr.CreatedAt.Format(time.RFC3339))
			assert.NoError(t, err)
			err = bwrRedis.DeleteFromHash(redisHashCacheHashAccessTime, test.ubr.Hash)
			assert.NoError(t, err)
			got, _ := bwrRedis.GetFromHash(redisHashCacheHashAccessTime, test.ubr.Hash)
			if !reflect.DeepEqual(test.want, got.(string)) {
				t.Errorf("Delete want %v | got %v", test.want, got)
			}
		})
	}
}

func Test_GetFromHash(t *testing.T) {
	bwrRedis.Client.FlushDB()
	ubr := mockUBR()
	val := ubr.CreatedAt.Format(time.RFC3339)
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want string
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: val,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bwrRedis.SetToHash(redisHashCacheHashAccessTime, test.ubr.Hash, test.ubr.CreatedAt.Format(time.RFC3339))
			assert.NoError(t, err)
			got, _ := bwrRedis.GetFromHash(redisHashCacheHashAccessTime, test.ubr.Hash)
			assert.Equal(t, test.want, got)
		})
	}
}

func Test_SetToHash(t *testing.T) {
	bwrRedis.Client.FlushDB()
	ubr := mockUBR()
	val := ubr.CreatedAt.Format(time.RFC3339)
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want string
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: val,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, _ := bwrRedis.GetFromHash(redisHashCacheHashAccessTime, test.ubr.Hash)
			assert.Equal(t, "", got)
			err := bwrRedis.SetToHash(redisHashCacheHashAccessTime, test.ubr.Hash, test.ubr.CreatedAt.Format(time.RFC3339))
			assert.NoError(t, err)
			got, _ = bwrRedis.GetFromHash(redisHashCacheHashAccessTime, test.ubr.Hash)
			assert.Equal(t, test.want, got)
		})
	}
}

func Test_DeleteFromSorted(t *testing.T) {
	bwrRedis.Client.FlushDB()
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want string
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bwrRedis.SetToSorted(redisSortedSetUnmovedBlock, float64(test.ubr.CreatedAt.UnixMicro()), test.ubr.Hash)
			assert.NoError(t, err)
			err = bwrRedis.DeleteFromSorted(redisSortedSetUnmovedBlock, test.ubr.Hash)
			assert.NoError(t, err)
			got, _ := bwrRedis.GetFromHash(redisSortedSetUnmovedBlock, test.ubr.Hash)
			if !reflect.DeepEqual(test.want, got.(string)) {
				t.Errorf("Delete want %v | got %v", test.want, got)
			}
		})
	}
}

func Test_GetCountFromSorted(t *testing.T) {
	bwrRedis.Client.FlushDB()
	count := 5
	for i := 0; i < count; i++ {
		ubr := mockUBR()
		err := bwrRedis.SetToSorted(redisSortedSetUnmovedBlock, float64(ubr.CreatedAt.UnixMicro()), ubr.Hash)
		if err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want int64
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: int64(count),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			num, err := bwrRedis.GetCountFromSorted(redisSortedSetUnmovedBlock)
			assert.NoError(t, err)
			assert.Equal(t, test.want, num)
		})
	}
}

func Test_GetRangeFromSorted(t *testing.T) {
	bwrRedis.Client.FlushDB()
	count := 5
	numRange := 3
	var testData []string

	for i := 0; i < count; i++ {
		ubr := mockUBR()
		err := bwrRedis.SetToSorted(redisSortedSetUnmovedBlock, float64(ubr.CreatedAt.UnixMicro()), ubr.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if i < numRange {
			testData = append(testData, ubr.Hash)
		}
	}

	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want []string
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: testData,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			values, err := bwrRedis.GetRangeFromSorted(redisSortedSetUnmovedBlock, 0, 2)
			assert.NoError(t, err)
			assert.Equal(t, test.want, values)
		})
	}
}

func Test_GetRangeByScoreFromSorted(t *testing.T) {
	bwrRedis.Client.FlushDB()
	count := 5
	numRange := 3
	var testData []*UnmovedBlockRecord

	for i := 0; i < count; i++ {
		ubr := mockUBR()
		err := bwrRedis.SetToSorted(redisSortedSetUnmovedBlock, float64(ubr.CreatedAt.UnixMicro()), ubr.Hash)
		if err != nil {
			t.Fatal(err)
		}
		if i < numRange {
			testData = append(testData, ubr)
		}
	}

	tests := []struct {
		name      string
		ubr       *UnmovedBlockRecord
		lastBlock int64
		count     int64
		want      []*UnmovedBlockRecord
	}{
		{
			name:      "OK",
			ubr:       mockUBR(),
			lastBlock: time.Now().UnixMicro(),
			count:     3,
			want:      testData,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := bwrRedis.GetRangeByScoreFromSorted(redisSortedSetUnmovedBlock, test.lastBlock, test.count)
			for idx := range got {
				assert.Equal(t, test.want[idx].Hash, got[idx].Hash)
				assert.Equal(t, test.want[idx].CreatedAt.UnixMicro(), got[idx].CreatedAt.UnixMicro())
			}
		})
	}
}

func Test_SetToSorted(t *testing.T) {
	bwrRedis.Client.FlushDB()
	tests := []struct {
		name string
		ubr  *UnmovedBlockRecord
		want string
	}{
		{
			name: "OK",
			ubr:  mockUBR(),
			want: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := bwrRedis.SetToSorted(redisSortedSetUnmovedBlock, float64(test.ubr.CreatedAt.UnixMicro()), test.ubr.Hash)
			assert.NoError(t, err)
			got, _ := bwrRedis.GetFromHash(redisSortedSetUnmovedBlock, test.ubr.Hash)
			if !reflect.DeepEqual(test.want, got.(string)) {
				t.Errorf("Delete want %v | got %v", test.want, got)
			}
		})
	}
}

func Test_Exec(t *testing.T) {
	bwrRedis.Client.FlushDB()
	t.Run("Ok", func(t *testing.T) {
		err := bwrRedis.Exec()
		if assert.Error(t, err) {
			assert.Equal(t, "ERR EXEC without MULTI", err.Error())
		}
	})
}

func Test_StartTx(t *testing.T) {
	bwrRedis.Client.FlushDB()
	t.Run("Ok", func(t *testing.T) {
		err := bwrRedis.StartTx()
		assert.NoError(t, err)
	})
}
